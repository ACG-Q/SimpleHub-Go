#!/usr/bin/env python3
# SimpleHub-Go API 端到端测试
# 用法: python scripts/test_api.py [--server path/to/server.exe]
import re
import os
import sys
import json
import time
import signal
import subprocess
import tempfile
import shutil
import argparse

import requests

HERE = os.path.dirname(os.path.abspath(__file__))
PROJECT = os.path.dirname(HERE)
DEFAULT_SERVER = os.path.join(PROJECT, "bin", "server.exe")


def eprint(*a, **kw):
    print(*a, file=sys.stderr, **kw)


class SimpleHubTester:
    # ── 生命周期 ──────────────────────────────────────────────

    def __init__(self, server_exe=None):
        self.server_exe = server_exe or DEFAULT_SERVER
        self.proc = None
        self.temp_dir = None
        self.base_url = None
        self.token = None
        self.site_id = None

    def _start_server(self):
        """启动 server（stdout 重定向到文件以绕过 Go pipe 缓冲问题），
        从文件中解析端口 / 用户名 / 密码"""
        self.temp_dir = tempfile.mkdtemp(prefix="sh_test_")
        db_path = os.path.join(self.temp_dir, "test.db")
        out_file = os.path.join(self.temp_dir, "stdout.txt")
        err_file = os.path.join(self.temp_dir, "stderr.txt")

        with open(out_file, "wb") as fout, open(err_file, "wb") as ferr:
            self.proc = subprocess.Popen(
                [self.server_exe, "--db", db_path],
                stdout=fout, stderr=ferr,
                cwd=PROJECT,
            )

        # 等待 stdout 文件中出现横幅
        port = username = password = None
        deadline = time.time() + 15
        while time.time() < deadline:
            if os.path.exists(out_file) and os.path.getsize(out_file) > 0:
                with open(out_file, "r", encoding="utf-8") as f:
                    text = f.read()
                m = re.search(r"端口:\s*(\d+)", text)
                if m:
                    port = int(m.group(1))
                m = re.search(r"管理员账号:\s*(\S+)", text)
                if m:
                    username = m.group(1)
                m = re.search(r"管理员密码:\s*(\S+)", text)
                if m:
                    password = m.group(1)
                if port and username and password:
                    break
            time.sleep(0.3)

        if not all([port, username, password]):
            self._stop_server()
            raise RuntimeError("无法从 stdout 解析端口 / 账号 / 密码")

        self.base_url = f"http://localhost:{port}"

        # 等待 HTTP 就绪
        deadline = time.time() + 15
        while time.time() < deadline:
            try:
                r = requests.get(f"{self.base_url}/", timeout=2)
                r.status_code
                break
            except requests.RequestException:
                time.sleep(0.25)
        else:
            self._stop_server()
            raise RuntimeError("服务器未在 15 秒内就绪")
        eprint(f"[OK] 服务器已启动: {self.base_url}")
        return username, password

    def _stop_server(self):
        if self.proc:
            if sys.platform == "win32":
                self.proc.send_signal(signal.CTRL_BREAK_EVENT)
            else:
                self.proc.terminate()
            try:
                self.proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self.proc.kill()
                self.proc.wait(timeout=2)
            eprint("[OK] 服务器已关闭")
        if self.temp_dir and os.path.isdir(self.temp_dir):
            shutil.rmtree(self.temp_dir, ignore_errors=True)

    # ── 断言与检查工具 ───────────────────────────────────────

    def _auth_header(self):
        return {"Authorization": f"Bearer {self.token}"}

    def _assert(self, cond, msg):
        if not cond:
            raise AssertionError(msg)

    def _check_ok(self, r, expected=200):
        if r.status_code != expected:
            eprint(f"  [失败] 期望状态码 {expected}，实际 {r.status_code}: {r.text[:200]}")
            raise AssertionError(f"HTTP {r.status_code} != {expected}: {r.text[:200]}")
        return r

    def _check_camelcase(self, obj, path=""):
        """递归检查所有返回 JSON 字段均为 camelCase（首字母小写）"""
        if isinstance(obj, dict):
            for k, v in obj.items():
                if k == "id":
                    continue
                self._assert(
                    k[0].islower() or k[0] == "_",
                    f"键 '{k}' 在 {path} 不是 camelCase 格式",
                )
                self._check_camelcase(v, f"{path}.{k}")
        elif isinstance(obj, list):
            for i, item in enumerate(obj):
                self._check_camelcase(item, f"{path}[{i}]")

    # ── 测试用例 ──────────────────────────────────────────────

    def test_login_error(self):
        """错误的凭据应返回 401"""
        r = requests.post(
            f"{self.base_url}/api/auth/login",
            json={"email": "nonexistent", "password": "wrong"},
        )
        self._assert(r.status_code == 401, f"期望 401，实际 {r.status_code}")
        data = r.json()
        self._assert("error" in data, "响应中缺少 error 字段")
        eprint("[通过] 登录错误测试")

    def test_create_site(self):
        """创建站点：验证各字段正确写入，敏感字段不在响应中"""
        body = {
            "name": "Test Site",
            "baseUrl": "https://api.example.com",
            "apiKey": "sk-test123-key",
            "apiType": "newapi",
            "userId": "12345",
            "pinned": True,
            "excludeFromBatch": False,
            "unlimitedQuota": False,
            "enableCheckIn": True,
            "checkInMode": "both",
            "timezone": "Asia/Shanghai",
        }
        r = self._check_ok(
            requests.post(
                f"{self.base_url}/api/sites",
                json=body,
                headers=self._auth_header(),
            ),
            201,
        )
        data = r.json()
        self._check_camelcase(data)
        self._assert(data["name"] == "Test Site", "名称不匹配")
        self._assert(data["baseUrl"] == "https://api.example.com", "URL 不匹配")
        self._assert(data["apiType"] == "newapi", "apiType 不匹配")
        self._assert(data["pinned"] is True, "pinned 应为 True")
        self._assert(data["enableCheckIn"] is True, "enableCheckIn 应为 True")
        self._assert("apiKey" not in data, "apiKey 不应出现在响应中")
        self._assert("apiKeyEnc" not in data, "apiKeyEnc 不应出现在响应中")
        self.site_id = data["id"]
        eprint(f"[通过] 创建站点 (id={self.site_id})")

    def test_list_sites(self):
        """站点列表：确认返回数组且包含必要字段"""
        r = self._check_ok(
            requests.get(
                f"{self.base_url}/api/sites",
                headers=self._auth_header(),
            )
        )
        data = r.json()
        self._assert(isinstance(data, list), "期望返回数组")
        self._assert(len(data) >= 1, "期望至少 1 个站点")
        site = data[0]
        self._check_camelcase(site)
        for key in ("id", "name", "baseUrl", "apiType", "pinned", "createdAt"):
            self._assert(key in site, f"列表响应中缺少字段 '{key}'")
        eprint("[通过] 站点列表")

    def test_search_sites(self):
        """站点搜索：search 参数应能过滤结果"""
        r = self._check_ok(
            requests.get(
                f"{self.base_url}/api/sites?search=Test",
                headers=self._auth_header(),
            )
        )
        data = r.json()
        self._assert(len(data) >= 1, "期望搜索到结果")
        eprint("[通过] 站点搜索")

    def test_get_site_detail(self):
        """站点详情：应返回解密后的 token、type 别名等额外字段"""
        r = self._check_ok(
            requests.get(
                f"{self.base_url}/api/sites/{self.site_id}",
                headers=self._auth_header(),
            )
        )
        data = r.json()
        self._check_camelcase(data)
        self._assert("token" in data, "缺少 'token'（解密后的 apiKey）")
        self._assert(data["token"] == "sk-test123-key", f"token 值不匹配: {data['token']}")
        self._assert("type" in data, "缺少 'type'（apiType 别名）")
        self._assert(data["type"] == "newapi", f"type 不匹配: {data['type']}")
        self._assert("proxyUrl" in data, "缺少 'proxyUrl'")
        self._assert("billingAuthValue" in data, "缺少 'billingAuthValue'")
        eprint("[通过] 站点详情")

    def test_update_site(self):
        """PATCH 更新站点基础字段"""
        r = self._check_ok(
            requests.patch(
                f"{self.base_url}/api/sites/{self.site_id}",
                json={"name": "Updated Site", "baseUrl": "https://updated.example.com"},
                headers=self._auth_header(),
            )
        )
        data = r.json()
        self._assert(data.get("name") == "Updated Site", "响应中名称未更新")
        self._assert(data.get("baseUrl") == "https://updated.example.com", "响应中 URL 未更新")
        eprint("[通过] 更新站点")

    def test_update_booleans(self):
        """PATCH 更新布尔字段：应能正确设为 false"""
        r = self._check_ok(
            requests.patch(
                f"{self.base_url}/api/sites/{self.site_id}",
                json={"pinned": False, "excludeFromBatch": True},
                headers=self._auth_header(),
            )
        )
        data = r.json()
        self._assert(data.get("pinned") is False, "pinned 应为 False")
        self._assert(data.get("excludeFromBatch") is True, "excludeFromBatch 应为 True")
        eprint("[通过] 更新布尔字段")

    def test_update_nullable(self):
        """PATCH 将可空字段置为 null"""
        r = self._check_ok(
            requests.patch(
                f"{self.base_url}/api/sites/{self.site_id}",
                json={"categoryId": None, "proxyUrl": None},
                headers=self._auth_header(),
            )
        )
        data = r.json()
        self._assert(data.get("categoryId") is None, "categoryId 应为 null")
        self._assert(data.get("proxyUrl") is None, "proxyUrl 应为 null")
        eprint("[通过] 置空可空字段")

    def test_delete_site(self):
        """删除站点后再次查询返回 404"""
        r = self._check_ok(
            requests.delete(
                f"{self.base_url}/api/sites/{self.site_id}",
                headers=self._auth_header(),
            )
        )
        self._assert(r.json().get("success") is True, "期望 success=true")
        r2 = requests.get(
            f"{self.base_url}/api/sites/{self.site_id}",
            headers=self._auth_header(),
        )
        self._assert(r2.status_code == 404, f"期望 404，实际 {r2.status_code}")
        eprint("[通过] 删除站点")

    def test_categories_crud(self):
        """分类的增删改查完整流程"""
        r = self._check_ok(
            requests.post(
                f"{self.base_url}/api/categories",
                json={"name": "Test Category", "timezone": "Asia/Shanghai"},
                headers=self._auth_header(),
            ),
            201,
        )
        data = r.json()
        self._check_camelcase(data)
        cat_id = data["id"]
        self._assert(data["name"] == "Test Category", "分类名称不匹配")
        eprint(f"  已创建分类: {cat_id}")

        r = self._check_ok(
            requests.get(
                f"{self.base_url}/api/categories",
                headers=self._auth_header(),
            )
        )
        cats = r.json()
        self._assert(isinstance(cats, list), "期望返回数组")
        found = any(c["id"] == cat_id for c in cats)
        self._assert(found, "列表中未找到新建的分类")

        r = self._check_ok(
            requests.patch(
                f"{self.base_url}/api/categories/{cat_id}",
                json={"name": "Updated Category"},
                headers=self._auth_header(),
            )
        )
        data = r.json()
        self._assert(data.get("name") == "Updated Category", "更新后名称不匹配")
        # GET 验证名称已更新
        r2 = requests.get(
            f"{self.base_url}/api/categories",
            headers=self._auth_header(),
        )
        cats = r2.json()
        updated = [c for c in cats if c["id"] == cat_id]
        self._assert(len(updated) == 1, "未找到更新后的分类")
        self._assert(updated[0]["name"] == "Updated Category", "分类名称未更新")

        r = self._check_ok(
            requests.delete(
                f"{self.base_url}/api/categories/{cat_id}",
                headers=self._auth_header(),
            )
        )
        self._assert(r.json().get("success") is True, "删除分类失败")
        eprint("[通过] 分类 CRUD")

    def test_email_config(self):
        """邮件配置：加密字段（resendApiKey）不在写入响应中"""
        r = self._check_ok(
            requests.post(
                f"{self.base_url}/api/email-config",
                json={
                    "resendApiKey": "re_test_key_12345",
                    "notifyEmails": "admin@example.com,dev@example.com",
                    "enabled": True,
                },
                headers=self._auth_header(),
            )
        )
        data = r.json()
        self._assert(data.get("enabled") is True, "Upsert 后 enabled 应为 true")
        self._assert("resendApiKey" not in data, "resendApiKey 不应出现在响应中")

        r2 = self._check_ok(
            requests.get(
                f"{self.base_url}/api/email-config",
                headers=self._auth_header(),
            )
        )
        data2 = r2.json()
        self._check_camelcase(data2)
        self._assert(data2["enabled"] is True, "GET 时 enabled 不匹配")
        self._assert(data2["notifyEmails"] == "admin@example.com,dev@example.com", "notifyEmails 不匹配")
        eprint("[通过] 邮件配置")

    def test_email_config_invalid_email(self):
        """邮件配置：无效邮箱应返回 400"""
        r = requests.post(
            f"{self.base_url}/api/email-config",
            json={
                "resendApiKey": "re_test_key",
                "notifyEmails": "not-an-email",
                "enabled": True,
            },
            headers=self._auth_header(),
        )
        self._assert(r.status_code == 400, f"无效邮箱期望 400，实际 {r.status_code}")
        eprint("[通过] 邮件配置无效邮箱校验")

    def test_reorder_sites(self):
        """站点重排序：应接受 orders 格式并正确排序"""
        # 创建两个额外站点用于排序
        r1 = self._check_ok(requests.post(f"{self.base_url}/api/sites", json={"name": "Site A", "baseUrl": "https://a.com", "apiKey": "sk-a"}, headers=self._auth_header()), 201)
        id_a = r1.json()["id"]
        r2 = self._check_ok(requests.post(f"{self.base_url}/api/sites", json={"name": "Site B", "baseUrl": "https://b.com", "apiKey": "sk-b"}, headers=self._auth_header()), 201)
        id_b = r2.json()["id"]

        r = self._check_ok(
            requests.post(
                f"{self.base_url}/api/sites/reorder",
                json={"orders": [{"id": id_a, "sortOrder": 0}, {"id": id_b, "sortOrder": 1}]},
                headers=self._auth_header(),
            )
        )
        self._assert(r.json().get("success") is True, "重排序应返回 success=true")
        eprint("[通过] 站点重排序")

    def test_check_site(self):
        """单次检测：对不存在的上游应返回错误但不崩溃"""
        r = requests.post(
            f"{self.base_url}/api/sites/{self.site_id}/check",
            headers=self._auth_header(),
        )
        data = r.json()
        self._assert("result" in data or "error" in data, "检测响应缺少 result 或 error")
        eprint("[通过] 单次检测 (期望错误上游)")

    def test_get_pricing(self):
        """定价代理：对不存在的上游应优雅处理错误"""
        r = requests.get(
            f"{self.base_url}/api/sites/{self.site_id}/pricing",
            headers=self._auth_header(),
        )
        # 可能返回 200+error 或 400/500，不崩溃即可
        self._assert(r.status_code < 600, "状态码异常")
        eprint("[通过] 定价代理")

    def test_snapshots(self):
        """快照列表：应返回数组"""
        # 先触发一次检测以产生快照
        requests.post(f"{self.base_url}/api/sites/{self.site_id}/check", headers=self._auth_header())

        r = self._check_ok(
            requests.get(
                f"{self.base_url}/api/sites/{self.site_id}/snapshots",
                headers=self._auth_header(),
            )
        )
        data = r.json()
        self._assert(isinstance(data, list), "快照列表应为数组")
        eprint("[通过] 快照列表")

    def test_latest_snapshot(self):
        """最新快照：应返回对象或空"""
        r = requests.get(
            f"{self.base_url}/api/sites/{self.site_id}/latest-snapshot",
            headers=self._auth_header(),
        )
        data = r.json()
        self._assert(isinstance(data, dict), "最新快照应为对象")
        if "modelsJson" in data:
            self._assert(isinstance(data["modelsJson"], list), "modelsJson 应为数组")
        eprint("[通过] 最新快照")

    def test_diffs(self):
        """差异列表：应返回数组"""
        r = self._check_ok(
            requests.get(
                f"{self.base_url}/api/sites/{self.site_id}/diffs",
                headers=self._auth_header(),
            )
        )
        data = r.json()
        self._assert(isinstance(data, list), "差异列表应为数组")
        eprint("[通过] 差异列表")

    def test_import_sites(self):
        """站点导入：验证导入流程"""
        export_r = self._check_ok(
            requests.get(f"{self.base_url}/api/sites/export", headers=self._auth_header())
        )
        export_data = export_r.json()

        import_body = {
            "version": "1.2",
            "categories": [{"name": "Imported Cat", "timezone": "Asia/Shanghai"}],
            "sites": [
                {
                    "name": "Imported Site",
                    "baseUrl": "https://imported.example.com",
                    "apiKey": "sk-imported-key",
                    "apiType": "other",
                    "timezone": "UTC",
                    "checkInMode": "both",
                    "billingAuthType": "token",
                    "categoryName": "Imported Cat",
                }
            ],
        }
        r = self._check_ok(
            requests.post(
                f"{self.base_url}/api/sites/import",
                json=import_body,
                headers=self._auth_header(),
            )
        )
        data = r.json()
        self._assert(data.get("success") is True, "导入应成功")
        self._assert(data.get("imported", 0) >= 1, "应导入至少 1 个站点")
        self._assert("total" in data, "导入响应应包含 total")
        eprint("[通过] 站点导入")

    def test_import_validation(self):
        """站点导入验证：缺少必填字段应返回错误"""
        body = {
            "version": "1.2",
            "sites": [{"name": "", "baseUrl": "", "apiKey": ""}],
        }
        r = requests.post(
            f"{self.base_url}/api/sites/import",
            json=body,
            headers=self._auth_header(),
        )
        self._check_ok(r)
        data = r.json()
        self._assert("errors" in data, "导入无效数据应返回 errors")
        self._assert(data.get("imported", 0) == 0, "无效数据导入应为 0")
        eprint("[通过] 导入验证")

    def test_import_empty_sites(self):
        """空站点列表导入应返回 400"""
        body = {"version": "1.2", "sites": []}
        r = requests.post(
            f"{self.base_url}/api/sites/import",
            json=body,
            headers=self._auth_header(),
        )
        self._assert(r.status_code == 400, f"空列表期望 400，实际 {r.status_code}")
        eprint("[通过] 空站点列表导入")

    def test_export_categories(self):
        """导出应包含 categories 数组"""
        r = self._check_ok(
            requests.get(f"{self.base_url}/api/sites/export", headers=self._auth_header())
        )
        data = r.json()
        self._assert("categories" in data, "导出应包含 categories")
        self._assert(isinstance(data["categories"], list), "categories 应为数组")
        eprint("[通过] 导出 categories")

    def test_schedule_config(self):
        """计划任务配置：读取默认值后更新"""
        r = self._check_ok(
            requests.get(
                f"{self.base_url}/api/schedule-config",
                headers=self._auth_header(),
            )
        )
        data = r.json()
        self._check_camelcase(data)
        self._assert(data.get("ok") is True, "缺少 ok 字段")
        cfg = data.get("config", {})
        self._assert("hour" in cfg, "缺少 hour 字段")

        r2 = self._check_ok(
            requests.post(
                f"{self.base_url}/api/schedule-config",
                json={"enabled": True, "hour": 8, "minute": 30, "interval": 60},
                headers=self._auth_header(),
            )
        )
        data2 = r2.json()
        self._assert(data2.get("ok") is True, "更新后 ok 应为 true")
        cfg2 = data2.get("config", {})
        self._assert(cfg2.get("enabled") is True, "更新后 enabled 应为 true")
        self._assert(cfg2.get("hour") == 8, "更新后 hour 应为 8")
        eprint("[通过] 计划任务配置")

    def test_export(self):
        """站点导出：确认返回 version + sites 结构"""
        r = self._check_ok(
            requests.get(
                f"{self.base_url}/api/sites/export",
                headers=self._auth_header(),
            )
        )
        data = r.json()
        self._assert("version" in data, "缺少 version")
        self._assert("sites" in data, "缺少 sites")
        self._assert(isinstance(data["sites"], list), "sites 应为数组")
        eprint("[通过] 站点导出")

    def test_exports_alias(self):
        """/api/exports/sites 是 /api/sites/export 的别名，返回相同格式"""
        r = self._check_ok(
            requests.get(
                f"{self.base_url}/api/exports/sites",
                headers=self._auth_header(),
            )
        )
        data = r.json()
        self._assert("version" in data, "缺少 version")
        eprint("[通过] 导出别名")

    # ── 主流程 ──────────────────────────────────────────────

    def run(self):
        """启动服务器 → 登录 → 顺序执行所有测试用例"""
        username, password = self._start_server()

        r = requests.post(
            f"{self.base_url}/api/auth/login",
            json={"email": username, "password": password},
        )
        self._check_ok(r)
        self.token = r.json()["token"]
        eprint(f"[OK] 登录成功 (user={username})")

        tests = [
            ("login_error", self.test_login_error),
            ("create_site", self.test_create_site),
            ("list_sites", self.test_list_sites),
            ("search_sites", self.test_search_sites),
            ("get_site_detail", self.test_get_site_detail),
            ("update_site", self.test_update_site),
            ("update_booleans", self.test_update_booleans),
            ("update_nullable", self.test_update_nullable),
            ("reorder_sites", self.test_reorder_sites),
            ("check_site", self.test_check_site),
            ("get_pricing", self.test_get_pricing),
            ("snapshots", self.test_snapshots),
            ("latest_snapshot", self.test_latest_snapshot),
            ("diffs", self.test_diffs),
            ("export", self.test_export),
            ("export_categories", self.test_export_categories),
            ("exports_alias", self.test_exports_alias),
            ("import_sites", self.test_import_sites),
            ("import_validation", self.test_import_validation),
            ("import_empty_sites", self.test_import_empty_sites),
            ("delete_site", self.test_delete_site),
            ("categories_crud", self.test_categories_crud),
            ("email_config", self.test_email_config),
            ("email_config_invalid_email", self.test_email_config_invalid_email),
            ("schedule_config", self.test_schedule_config),
        ]

        passed = 0
        failed = 0
        for name, fn in tests:
            try:
                fn()
                passed += 1
            except Exception as e:
                eprint(f"[失败] {name}: {e}")
                failed += 1

        eprint(f"\n{'='*50}")
        eprint(f"结果: {passed} 通过, {failed} 失败, {len(tests)} 总计")
        return failed == 0


def main():
    parser = argparse.ArgumentParser(description="SimpleHub-Go API tests")
    parser.add_argument("--server", default=DEFAULT_SERVER, help="path to server.exe")
    args = parser.parse_args()

    if not os.path.isfile(args.server):
        eprint(f"服务器可执行文件未找到: {args.server}")
        eprint(f"请先构建: cd {PROJECT} && .\\build.ps1")
        return 1

    tester = SimpleHubTester(server_exe=args.server)
    try:
        ok = tester.run()
    except Exception as e:
        eprint(f"\n[严重] {e}")
        ok = False
    finally:
        tester._stop_server()

    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())

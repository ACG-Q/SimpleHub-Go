export const SITE_TYPES = {
  NEWAPI: 'newapi',
  VELOERA: 'veloera',
  DONEHUB: 'donehub',
  VOAPI: 'voapi',
  OTHER: 'other',
}

export const SITE_TYPES_LIST = Object.values(SITE_TYPES)

export const SITE_TYPE_DESCS = {
  [SITE_TYPES.NEWAPI]: 'New API 中转站，支持自动签到和余额监控',
  [SITE_TYPES.VELOERA]: 'Veloera 中转站，支持自动签到和余额监控',
  [SITE_TYPES.DONEHUB]: 'DoneHub 中转站，支持令牌管理',
  [SITE_TYPES.VOAPI]: 'VOAPI 中转站，支持自动签到和余额监控',
  [SITE_TYPES.OTHER]: '通用中转站，支持自定义 Billing 配置',
}

export const SITE_TYPE_LABELS = {
  [SITE_TYPES.NEWAPI]: 'New API',
  [SITE_TYPES.VELOERA]: 'Veloera',
  [SITE_TYPES.DONEHUB]: 'DoneHub',
  [SITE_TYPES.VOAPI]: 'VOAPI',
  [SITE_TYPES.OTHER]: '通用',
}

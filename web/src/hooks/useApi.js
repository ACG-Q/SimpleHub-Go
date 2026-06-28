import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import { API } from '../constants/api'

export function useSites(search = '') {
  return useQuery({
    queryKey: ['sites', search],
    queryFn: () => api.get(search ? `${API.SITES}?search=${encodeURIComponent(search)}` : API.SITES),
    placeholderData: (prev) => prev,
  })
}

export function useSite(id) {
  return useQuery({
    queryKey: ['site', id],
    queryFn: () => api.get(`/api/sites/${id}`),
    enabled: !!id,
  })
}

export function useCategories() {
  return useQuery({
    queryKey: ['categories'],
    queryFn: () => api.get(API.CATEGORIES),
    placeholderData: (prev) => prev,
  })
}

export function useEmailConfig() {
  return useQuery({
    queryKey: ['email-config'],
    queryFn: () => api.get(API.EMAIL_CONFIG),
  })
}

export function useScheduleConfig() {
  return useQuery({
    queryKey: ['schedule-config'],
    queryFn: () => api.get(API.SCHEDULE_CONFIG),
  })
}

export function useSitesDiffs(id) {
  return useQuery({
    queryKey: ['site-diffs', id],
    queryFn: () => api.get(`/api/sites/${id}/diffs?limit=20`),
    enabled: !!id,
  })
}

export function useSitesSnapshots(id) {
  return useQuery({
    queryKey: ['site-snapshots', id],
    queryFn: () => api.get(`/api/sites/${id}/snapshots?limit=1`),
    enabled: !!id,
  })
}

export function usePricing(id) {
  return useQuery({
    queryKey: ['pricing', id],
    queryFn: () => api.get(`/api/sites/${id}/pricing`),
    enabled: !!id,
    retry: 1,
  })
}

export function useTokens(siteId) {
  return useQuery({
    queryKey: ['tokens', siteId],
    queryFn: () => api.get(`/api/sites/${siteId}/tokens`),
    enabled: !!siteId,
    select: (data) => {
      let list = []
      if (Array.isArray(data)) list = data
      else if (data?.data?.items) list = data.data.items
      else if (data?.data?.data) list = data.data.data
      else if (data?.data) list = data.data
      return list
    },
  })
}

export function useGroups(siteId) {
  return useQuery({
    queryKey: ['groups', siteId],
    queryFn: () => api.get(`/api/sites/${siteId}/groups`),
    enabled: !!siteId,
    select: (data) => {
      if (data?.success && data?.data) {
        return Object.entries(data.data).map(([value, g]) => ({ value, label: g.name || g.desc || value }))
      }
      return []
    },
  })
}

export function useCreateSite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data) => api.post(API.SITES, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['sites'] }),
  })
}

export function useUpdateSite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data) => api.patch(`/api/sites/${data.id}`, data),
    onSuccess: (_, vars) => {
      qc.invalidateQueries({ queryKey: ['sites'] })
      if (vars?.id) {
        qc.invalidateQueries({ queryKey: ['site', vars.id] })
        qc.invalidateQueries({ queryKey: ['groups', vars.id] })
      }
    },
  })
}

export function useDeleteSite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id) => api.del(`/api/sites/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['sites'] }),
  })
}

export function useCheckSite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, skipNotification = true }) => api.post(`/api/sites/${id}/check?skipNotification=${skipNotification}`),
    onSuccess: (data, { id }) => {
      qc.invalidateQueries({ queryKey: ['sites'] })
      qc.invalidateQueries({ queryKey: ['site', id] })
      qc.invalidateQueries({ queryKey: ['site-diffs', id] })
      qc.invalidateQueries({ queryKey: ['site-snapshots', id] })
      qc.invalidateQueries({ queryKey: ['pricing', id] })
      return data
    },
  })
}

export function useBatchCheckCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (categoryId) => api.post(API.CATEGORY_CHECK.replace(':id', categoryId)),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['sites'] }),
  })
}

export function useUpdateSortOrder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, sortOrder }) => api.patch(`/api/sites/${id}`, { sortOrder }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['sites'] }),
  })
}

export function useUpdateSchedule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, scheduleCron }) => api.patch(`/api/sites/${id}`, { scheduleCron }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['sites'] }),
  })
}

export function useCreateCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data) => api.post(API.CATEGORIES, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['categories'] }),
  })
}

export function useUpdateCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data) => api.put(API.CATEGORIES, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['categories'] }),
  })
}

export function useDeleteCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id) => api.del(`/api/categories/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['categories'] })
      qc.invalidateQueries({ queryKey: ['sites'] })
    },
  })
}

export function useSaveEmailConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data) => api.post(API.EMAIL_CONFIG, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['email-config'] }),
  })
}

export function useTestEmail() {
  return useMutation({
    mutationFn: () => api.post(API.EMAIL_CONFIG_TEST),
  })
}

export function useSaveScheduleConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data) => api.post(API.SCHEDULE_CONFIG, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['schedule-config'] }),
  })
}

export function useTriggerGlobalCheck() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post('/api/schedule-config/trigger'),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['sites'] }),
  })
}

export function useCreateToken(siteId) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload) => api.post(`/api/sites/${siteId}/tokens`, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tokens', siteId] })
      qc.invalidateQueries({ queryKey: ['groups', siteId] })
    },
  })
}

export function useUpdateToken(siteId) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload) => api.put(`/api/sites/${siteId}/tokens`, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tokens', siteId] })
      qc.invalidateQueries({ queryKey: ['groups', siteId] })
    },
  })
}

export function useDeleteToken(siteId) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (tokenId) => api.del(`/api/sites/${siteId}/tokens/${tokenId}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['tokens', siteId] })
      qc.invalidateQueries({ queryKey: ['groups', siteId] })
    },
  })
}

export function useLatestSnapshot(siteId) {
  return useQuery({
    queryKey: ['sites', siteId, 'latest-snapshot'],
    queryFn: () => api.get(`/api/sites/${siteId}/latest-snapshot`),
    enabled: !!siteId,
  })
}

export function useRevealTokenKey(siteId) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (tokenId) => api.post(`/api/sites/${siteId}/tokens/${tokenId}/key`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tokens', siteId] }),
  })
}

export function useDashboardStats() {
  return useQuery({
    queryKey: ['dashboard-stats'],
    queryFn: () => api.get(API.DASHBOARD_STATS),
  })
}

export function useRedeemCode(siteId) {
  return useMutation({
    mutationFn: (key) => api.post(`/api/sites/${siteId}/redeem`, { key }),
  })
}

export function useExportSites() {
  return useMutation({
    mutationFn: () => api.post(API.EXPORTS_SITES),
  })
}

export function useImportSites() {
  return useMutation({
    mutationFn: (data) => api.post(API.SITES_IMPORT, data),
  })
}

export function useLogin() {
  return useMutation({
    mutationFn: ({ username, password }) => api.post(API.AUTH, { email: username, password }),
  })
}

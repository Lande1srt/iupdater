import request from '../utils/request'

export function login(data) {
  return request.post('/login', data)
}

export function getApps(params) {
  return request.get('/admin/apps', { params })
}

export function getApp(id) {
  return request.get(`/admin/apps/${id}`)
}

export function createApp(data) {
  return request.post('/admin/apps', data)
}

export function updateApp(id, data) {
  return request.put(`/admin/apps/${id}`, data)
}

export function deleteApp(id) {
  return request.delete(`/admin/apps/${id}`)
}

export function checkStorageConnection(data) {
  return request.post('/admin/storage/check', data)
}

export function getVersions(params) {
  return request.get('/admin/versions', { params })
}

export function getVersion(id) {
  return request.get(`/admin/versions/${id}`)
}

export function createVersion(data) {
  return request.post('/admin/versions', data)
}

export function uploadVersionFile(formData, onProgress) {
  return request.post('/admin/versions/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 300000,
    onUploadProgress: onProgress
  })
}

export function probeFileInfo(url) {
  return request.post('/admin/versions/probe', { url })
}

export function updateVersionStatus(id, status) {
  return request.put(`/admin/versions/${id}/status`, { status })
}

export function deleteVersion(id) {
  return request.delete(`/admin/versions/${id}`)
}

export function getComponents(params) {
  return request.get('/admin/components', { params })
}

export function getComponent(id) {
  return request.get(`/admin/components/${id}`)
}

export function createComponent(data) {
  return request.post('/admin/components', data)
}

export function uploadComponentFile(formData, onProgress) {
  return request.post('/admin/components/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 300000,
    onUploadProgress: onProgress
  })
}

export function updateComponentStatus(id, status) {
  return request.put(`/admin/components/${id}/status`, { status })
}

export function deleteComponent(id) {
  return request.delete(`/admin/components/${id}`)
}

// ============ 存储配置档案 ============
export function getStorageProfiles(params) {
  return request.get('/admin/storage-profiles', { params })
}

export function getStorageProfile(id) {
  return request.get(`/admin/storage-profiles/${id}`)
}

export function createStorageProfile(data) {
  return request.post('/admin/storage-profiles', data)
}

export function updateStorageProfile(id, data) {
  return request.put(`/admin/storage-profiles/${id}`, data)
}

export function deleteStorageProfile(id) {
  return request.delete(`/admin/storage-profiles/${id}`)
}

export function checkStorageProfileConnection(id) {
  return request.post(`/admin/storage-profiles/${id}/check`)
}

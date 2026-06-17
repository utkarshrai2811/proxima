import { gqlClient } from './graphql'

// ---- Types (mirror pkg/api/schema.graphql) ---------------------------------

export type HttpMethod =
  | 'GET' | 'HEAD' | 'POST' | 'PUT' | 'DELETE'
  | 'CONNECT' | 'OPTIONS' | 'TRACE' | 'PATCH'

export type HttpProtocol = 'HTTP10' | 'HTTP11' | 'HTTP20'

export interface HttpHeader {
  key: string
  value: string
}

export interface HttpResponseLog {
  id: string
  proto: HttpProtocol
  statusCode: number
  statusReason: string
  body: string | null
  headers: HttpHeader[]
}

export interface HttpRequestLog {
  id: string
  url: string
  method: HttpMethod
  proto: string
  headers: HttpHeader[]
  body: string | null
  timestamp: string
  response: HttpResponseLog | null
}

export interface InterceptSettings {
  requestsEnabled: boolean
  responsesEnabled: boolean
  requestFilter: string | null
  responseFilter: string | null
}

export interface Project {
  id: string
  name: string
  isActive: boolean
  settings: { intercept: InterceptSettings }
}

export interface ScopeHeader {
  key: string | null
  value: string | null
}

export interface ScopeRule {
  url: string | null
  header: ScopeHeader | null
  body: string | null
}

export interface InterceptedRequest {
  id: string
  url: string
  method: HttpMethod
  proto: HttpProtocol
  headers: HttpHeader[]
  body: string | null
}

export interface SenderResponse {
  id: string
  proto: HttpProtocol
  statusCode: number
  statusReason: string
  body: string | null
  headers: HttpHeader[]
}

export interface SenderRequest {
  id: string
  url: string
  method: HttpMethod
  proto: HttpProtocol
  headers: HttpHeader[] | null
  body: string | null
  timestamp: string
  response: SenderResponse | null
}

// ---- Fragments -------------------------------------------------------------

const requestLogFields = `
  id
  url
  method
  proto
  timestamp
  headers { key value }
  body
  response {
    id
    proto
    statusCode
    statusReason
    headers { key value }
    body
  }
`

// ---- Proxy log -------------------------------------------------------------

export async function fetchRequestLogs(): Promise<HttpRequestLog[]> {
  const data = await gqlClient.request<{ httpRequestLogs: HttpRequestLog[] }>(`
    query { httpRequestLogs { ${requestLogFields} } }
  `)
  return data.httpRequestLogs
}

export async function fetchRequestLog(id: string): Promise<HttpRequestLog | null> {
  const data = await gqlClient.request<{ httpRequestLog: HttpRequestLog | null }>(
    `query ($id: ID!) { httpRequestLog(id: $id) { ${requestLogFields} } }`,
    { id },
  )
  return data.httpRequestLog
}

export async function clearRequestLogs(): Promise<void> {
  await gqlClient.request(`mutation { clearHTTPRequestLog { success } }`)
}

export interface RequestLogFilter {
  onlyInScope: boolean
  searchExpression: string | null
}

export async function fetchRequestLogFilter(): Promise<RequestLogFilter | null> {
  const data = await gqlClient.request<{ httpRequestLogFilter: RequestLogFilter | null }>(
    `query { httpRequestLogFilter { onlyInScope searchExpression } }`,
  )
  return data.httpRequestLogFilter
}

export async function setRequestLogFilter(
  searchExpression: string,
  onlyInScope: boolean,
): Promise<void> {
  await gqlClient.request(
    `mutation ($f: HttpRequestLogFilterInput) {
      setHttpRequestLogFilter(filter: $f) { onlyInScope searchExpression }
    }`,
    { f: { searchExpression: searchExpression || null, onlyInScope } },
  )
}

// ---- Projects --------------------------------------------------------------

export async function fetchActiveProject(): Promise<Project | null> {
  const data = await gqlClient.request<{ activeProject: Project | null }>(`
    query {
      activeProject {
        id name isActive
        settings { intercept { requestsEnabled responsesEnabled requestFilter responseFilter } }
      }
    }
  `)
  return data.activeProject
}

export async function fetchProjects(): Promise<Project[]> {
  const data = await gqlClient.request<{ projects: Project[] }>(`
    query {
      projects {
        id name isActive
        settings { intercept { requestsEnabled responsesEnabled requestFilter responseFilter } }
      }
    }
  `)
  return data.projects
}

export async function createProject(name: string): Promise<void> {
  await gqlClient.request(`mutation ($name: String!) { createProject(name: $name) { id } }`, { name })
}

export async function openProject(id: string): Promise<void> {
  await gqlClient.request(`mutation ($id: ID!) { openProject(id: $id) { id } }`, { id })
}

export async function closeProject(): Promise<void> {
  await gqlClient.request(`mutation { closeProject { success } }`)
}

export async function deleteProject(id: string): Promise<void> {
  await gqlClient.request(`mutation ($id: ID!) { deleteProject(id: $id) { success } }`, { id })
}

// ---- Scope -----------------------------------------------------------------

export async function fetchScope(): Promise<ScopeRule[]> {
  const data = await gqlClient.request<{ scope: ScopeRule[] }>(
    `query { scope { url header { key value } body } }`,
  )
  return data.scope
}

export async function setScope(rules: ScopeRule[]): Promise<void> {
  const scope = rules.map((r) => ({
    url: r.url || null,
    body: r.body || null,
    header:
      r.header && (r.header.key || r.header.value)
        ? { key: r.header.key || null, value: r.header.value || null }
        : null,
  }))
  await gqlClient.request(
    `mutation ($scope: [ScopeRuleInput!]!) { setScope(scope: $scope) { url } }`,
    { scope },
  )
}

// ---- Intercept -------------------------------------------------------------

export async function fetchInterceptedRequests(): Promise<InterceptedRequest[]> {
  const data = await gqlClient.request<{ interceptedRequests: InterceptedRequest[] }>(`
    query { interceptedRequests { id url method proto headers { key value } body } }
  `)
  return data.interceptedRequests
}

export async function modifyRequest(req: {
  id: string
  url: string
  method: HttpMethod
  proto: HttpProtocol
  headers: HttpHeader[]
  body: string | null
}): Promise<void> {
  await gqlClient.request(
    `mutation ($request: ModifyRequestInput!) { modifyRequest(request: $request) { success } }`,
    { request: req },
  )
}

export async function cancelRequest(id: string): Promise<void> {
  await gqlClient.request(`mutation ($id: ID!) { cancelRequest(id: $id) { success } }`, { id })
}

export async function updateInterceptSettings(s: {
  requestsEnabled: boolean
  responsesEnabled: boolean
}): Promise<void> {
  await gqlClient.request(
    `mutation ($input: UpdateInterceptSettingsInput!) {
      updateInterceptSettings(input: $input) { requestsEnabled responsesEnabled }
    }`,
    { input: { ...s, requestFilter: null, responseFilter: null } },
  )
}

// ---- HTTP client (sender) --------------------------------------------------

const senderRequestFields = `
  id
  url
  method
  proto
  timestamp
  headers { key value }
  body
  response { id proto statusCode statusReason headers { key value } body }
`

export async function createOrUpdateSenderRequest(req: {
  id?: string
  url: string
  method: HttpMethod
  headers: HttpHeader[]
  body: string | null
}): Promise<SenderRequest> {
  const data = await gqlClient.request<{ createOrUpdateSenderRequest: SenderRequest }>(
    `mutation ($request: SenderRequestInput!) {
      createOrUpdateSenderRequest(request: $request) { ${senderRequestFields} }
    }`,
    {
      request: {
        id: req.id ?? null,
        url: req.url,
        method: req.method,
        headers: req.headers,
        body: req.body,
      },
    },
  )
  return data.createOrUpdateSenderRequest
}

export async function sendSenderRequest(id: string): Promise<SenderRequest> {
  const data = await gqlClient.request<{ sendRequest: SenderRequest }>(
    `mutation ($id: ID!) { sendRequest(id: $id) { ${senderRequestFields} } }`,
    { id },
  )
  return data.sendRequest
}

export async function createSenderFromLog(id: string): Promise<SenderRequest> {
  const data = await gqlClient.request<{ createSenderRequestFromHttpRequestLog: SenderRequest }>(
    `mutation ($id: ID!) {
      createSenderRequestFromHttpRequestLog(id: $id) { ${senderRequestFields} }
    }`,
    { id },
  )
  return data.createSenderRequestFromHttpRequestLog
}

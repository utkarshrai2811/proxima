import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import ProxyLog from './pages/ProxyLog'
import Intercept from './pages/Intercept'
import Scope from './pages/Scope'
import HttpClient from './pages/HttpClient'
import Settings from './pages/Settings'
import WebSockets from './pages/WebSockets'
import Fuzzer from './pages/Fuzzer'
import Plugins from './pages/Plugins'

export default function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<ProxyLog />} />
        <Route path="/intercept" element={<Intercept />} />
        <Route path="/scope" element={<Scope />} />
        <Route path="/client" element={<HttpClient />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="/fuzzer" element={<Fuzzer />} />
        <Route path="/websockets" element={<WebSockets />} />
        <Route path="/plugins" element={<Plugins />} />
      </Routes>
    </Layout>
  )
}

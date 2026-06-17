import { Routes, Route } from 'react-router-dom'
import { Zap, Puzzle } from 'lucide-react'
import Layout from './components/Layout'
import StubPage from './components/StubPage'
import ProxyLog from './pages/ProxyLog'
import Intercept from './pages/Intercept'
import Scope from './pages/Scope'
import HttpClient from './pages/HttpClient'
import Settings from './pages/Settings'
import WebSockets from './pages/WebSockets'

export default function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<ProxyLog />} />
        <Route path="/intercept" element={<Intercept />} />
        <Route path="/scope" element={<Scope />} />
        <Route path="/client" element={<HttpClient />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="/fuzzer" element={<StubPage title="Fuzzer" icon={Zap} />} />
        <Route path="/websockets" element={<WebSockets />} />
        <Route path="/plugins" element={<StubPage title="Plugins" icon={Puzzle} />} />
      </Routes>
    </Layout>
  )
}

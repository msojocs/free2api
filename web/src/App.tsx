import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { App as AntdApp, ConfigProvider } from 'antd'
import enUS from 'antd/locale/en_US'
import zhCN from 'antd/locale/zh_CN'
import { useTranslation } from 'react-i18next'
import AppLayout from './components/AppLayout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import TaskList from './pages/TaskList'
import AccountList from './pages/AccountList'
import ProxyManager from './pages/ProxyManager'
import PushTemplateManager from './pages/PushTemplateManager'
import SystemSettings from './pages/SystemSettings'
import TempMailProviderManager from './pages/TempMailProviderManager'
import { useAuthStore } from './store/auth'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token)
  return token ? <>{children}</> : <Navigate to="/login" replace />
}

export default function App() {
  const { i18n } = useTranslation()
  const antdLocale = i18n.language.startsWith('zh') ? zhCN : enUS

  return (
    <ConfigProvider locale={antdLocale}>
      <AntdApp>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route
              path="/"
              element={
                <PrivateRoute>
                  <AppLayout />
                </PrivateRoute>
              }
            >
              <Route index element={<Navigate to="/dashboard" replace />} />
              <Route path="dashboard" element={<Dashboard />} />
              <Route path="tasks" element={<TaskList />} />
              <Route path="accounts" element={<AccountList />} />
              <Route path="proxies" element={<ProxyManager />} />
              <Route path="settings" element={<SystemSettings />} />
              <Route path="temp-mail-providers" element={<TempMailProviderManager />} />
              <Route path="push-templates" element={<PushTemplateManager />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </AntdApp>
    </ConfigProvider>
  )
}

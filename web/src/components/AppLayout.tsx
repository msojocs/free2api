import { Layout, Menu, Button, Typography, theme, Dropdown } from 'antd'
import {
  DashboardOutlined,
  UnorderedListOutlined,
  UserOutlined,
  GlobalOutlined,
  LogoutOutlined,
  SendOutlined,
  TranslationOutlined,
  InboxOutlined,
  SettingOutlined,
} from '@ant-design/icons'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '../store/auth'

const { Sider, Content, Header } = Layout
const { Text } = Typography

export default function AppLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { user, logout } = useAuthStore()
  const { token: designToken } = theme.useToken()
  const { t, i18n } = useTranslation()

  function handleLogout() {
    logout()
    navigate('/login')
  }

  const menuItems = [
    { key: '/dashboard', icon: <DashboardOutlined />, label: t('nav.dashboard') },
    { key: '/tasks', icon: <UnorderedListOutlined />, label: t('nav.tasks') },
    { key: '/accounts', icon: <UserOutlined />, label: t('nav.accounts') },
    { key: '/proxies', icon: <GlobalOutlined />, label: t('nav.proxies') },
    { key: '/temp-mail-providers', icon: <InboxOutlined />, label: t('nav.tempMailProviders') },
    { key: '/push-templates', icon: <SendOutlined />, label: t('nav.pushTemplates') },
    { key: '/settings', icon: <SettingOutlined />, label: t('nav.settings') },
  ]

  const langMenuItems = [
    {
      key: 'en',
      label: t('lang.en'),
      onClick: () => i18n.changeLanguage('en'),
    },
    {
      key: 'zh',
      label: t('lang.zh'),
      onClick: () => i18n.changeLanguage('zh'),
    },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        theme="dark"
        style={{
          display: 'flex',
          flexDirection: 'column',
          position: 'fixed',
          height: '100vh',
          left: 0,
          top: 0,
          bottom: 0,
        }}
      >
        <div
          style={{
            padding: '16px',
            textAlign: 'center',
            borderBottom: '1px solid rgba(255,255,255,0.1)',
          }}
        >
          <Text strong style={{ color: '#fff', fontSize: 18 }}>
            Free2API
          </Text>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ flex: 1, borderRight: 0 }}
        />
        <div
          style={{
            padding: '16px',
            borderTop: '1px solid rgba(255,255,255,0.1)',
          }}
        >
          <Text style={{ color: 'rgba(255,255,255,0.65)', display: 'block', marginBottom: 8 }}>
            {user?.username}
          </Text>
          <Button
            icon={<LogoutOutlined />}
            type="text"
            style={{ color: 'rgba(255,255,255,0.65)', paddingLeft: 0 }}
            onClick={handleLogout}
          >
            {t('nav.logout')}
          </Button>
        </div>
      </Sider>
      <Layout style={{ marginLeft: 200 }}>
        <Header
          style={{
            background: designToken.colorBgContainer,
            padding: '0 24px',
            borderBottom: `1px solid ${designToken.colorBorderSecondary}`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-end',
          }}
        >
          <Dropdown menu={{ items: langMenuItems, selectedKeys: [i18n.language.startsWith('zh') ? 'zh' : 'en'] }} trigger={['click']}>
            <Button icon={<TranslationOutlined />} type="text">
              {i18n.language.startsWith('zh') ? t('lang.zh') : t('lang.en')}
            </Button>
          </Dropdown>
        </Header>
        <Content style={{ padding: 24, background: designToken.colorBgLayout }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}

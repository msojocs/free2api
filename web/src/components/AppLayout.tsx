import { useEffect, useMemo, useState } from 'react'
import { Layout, Menu, Button, Typography, theme, Dropdown, Avatar, Space } from 'antd'
import {
  DashboardOutlined,
  UnorderedListOutlined,
  UserOutlined,
  GlobalOutlined,
  SendOutlined,
  TranslationOutlined,
  InboxOutlined,
  SettingOutlined,
  LockOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
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
  const siderWidth = 200
  const collapsedSiderWidth = 80
  const [collapsed, setCollapsed] = useState(false)
  const [broken, setBroken] = useState(false)

  function handleLogout() {
    logout()
    navigate('/login')
  }

  function getAvatarInitial(name: string | undefined): string {
    if (!name) return '?'
    const firstChar = Array.from(name.trim())[0]
    return (firstChar || '?').toUpperCase()
  }

  function getAvatarColor(seed: string | undefined): string {
    const text = seed || 'user'
    let hash = 0
    for (let i = 0; i < text.length; i += 1) {
      hash = text.charCodeAt(i) + ((hash << 5) - hash)
    }
    return `hsl(${Math.abs(hash) % 360}, 62%, 46%)`
  }

  const menuItems = [
    { key: '/dashboard', icon: <DashboardOutlined />, label: t('nav.dashboard') },
    { key: '/tasks', icon: <UnorderedListOutlined />, label: t('nav.tasks') },
    {
      key: '/accounts',
      icon: <UserOutlined />,
      label: t('nav.accounts'),
      children: [
        { key: '/accounts/chatgpt', label: t('nav.accountsChatgpt') },
        { key: '/accounts/cursor', label: t('nav.accountsCursor') },
        { key: '/accounts/all', label: t('nav.accountsAll') },
      ],
    },
    { key: '/proxies', icon: <GlobalOutlined />, label: t('nav.proxies') },
    { key: '/temp-mail-providers', icon: <InboxOutlined />, label: t('nav.tempMailProviders') },
    { key: '/push-templates', icon: <SendOutlined />, label: t('nav.pushTemplates') },
    { key: '/settings', icon: <SettingOutlined />, label: t('nav.settings') },
  ]

  const selectedMenuKey = useMemo(() => {
    if (location.pathname.startsWith('/accounts')) {
      if (location.pathname === '/accounts') return '/accounts/chatgpt'
      return location.pathname
    }
    return location.pathname
  }, [location.pathname])

  const [openMenuKeys, setOpenMenuKeys] = useState<string[]>([])

  useEffect(() => {
    if (location.pathname.startsWith('/accounts')) {
      setOpenMenuKeys((prev) => (prev.includes('/accounts') ? prev : ['/accounts']))
    }
  }, [location.pathname])

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

  const userMenuItems = [
    {
      key: 'change-password',
      icon: <LockOutlined />,
      label: t('userMenu.changePassword'),
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: t('nav.logout'),
    },
  ]

  const avatarStyle = useMemo(
    () => ({
      backgroundColor: getAvatarColor(user?.username),
      color: '#fff',
      cursor: 'pointer',
      userSelect: 'none' as const,
      textTransform: 'uppercase' as const,
    }),
    [user?.username],
  )

  function handleUserMenuClick({ key }: { key: string }) {
    if (key === 'logout') {
      handleLogout()
      return
    }
    if (key === 'change-password') {
      navigate('/change-password')
    }
  }

  const contentMarginLeft = broken ? 0 : (collapsed ? collapsedSiderWidth : siderWidth)

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        theme="dark"
        collapsible
        trigger={null}
        width={siderWidth}
        collapsedWidth={broken ? 0 : collapsedSiderWidth}
        collapsed={collapsed}
        breakpoint="lg"
        onBreakpoint={(isBroken) => {
          setBroken(isBroken)
          setCollapsed(isBroken)
        }}
        onCollapse={(value) => setCollapsed(value)}
        style={{
          display: 'flex',
          flexDirection: 'column',
          position: 'fixed',
          height: '100vh',
          left: 0,
          top: 0,
          bottom: 0,
          zIndex: 1000,
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
            {collapsed && !broken ? 'AAR' : 'AI Auto Register'}
          </Text>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedMenuKey]}
          openKeys={openMenuKeys}
          onOpenChange={(keys) => setOpenMenuKeys(keys as string[])}
          items={menuItems}
          onClick={({ key }) => {
            navigate(key)
            if (broken) {
              setCollapsed(true)
            }
          }}
          style={{ flex: 1, borderRight: 0 }}
        />
      </Sider>
      <Layout style={{ marginLeft: contentMarginLeft, transition: 'margin-left 0.2s' }}>
        <Header
          style={{
            background: designToken.colorBgContainer,
            padding: '0 24px',
            borderBottom: `1px solid ${designToken.colorBorderSecondary}`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed((prev) => !prev)}
            aria-label="toggle sidebar"
          />
          <Space size={8}>
            <Dropdown menu={{ items: langMenuItems, selectedKeys: [i18n.language.startsWith('zh') ? 'zh' : 'en'] }} trigger={['click']}>
              <Button icon={<TranslationOutlined />} type="text">
                {i18n.language.startsWith('zh') ? t('lang.zh') : t('lang.en')}
              </Button>
            </Dropdown>
            <Dropdown menu={{ items: userMenuItems, onClick: handleUserMenuClick }} trigger={['click']}>
              {user?.avatar_url ? (
                <Avatar src={user.avatar_url} style={{ cursor: 'pointer' }} />
              ) : (
                <Avatar style={avatarStyle}>{getAvatarInitial(user?.username)}</Avatar>
              )}
            </Dropdown>
          </Space>
        </Header>
        <Content style={{ padding: 24, background: designToken.colorBgLayout }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}

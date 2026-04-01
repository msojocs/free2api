import { useEffect, useState } from 'react'
import {
  App as AntdApp,
  Table,
  Button,
  Modal,
  Form,
  Input,
  Space,
  Typography,
  Popconfirm,
  Tag,
  Select,
  type TableProps,
} from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { getProxies, createProxy, deleteProxy, testProxy, type Proxy } from '../api/proxies'

const { Title } = Typography

const PROXY_PROTOCOLS = ['http', 'https', 'socks5']

export default function ProxyManager() {
  const [proxies, setProxies] = useState<Proxy[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [testingId, setTestingId] = useState<number | null>(null)
  const [form] = Form.useForm()
  const { t } = useTranslation()
  const { message } = AntdApp.useApp()

  async function fetchProxies() {
    setLoading(true)
    try {
      const { data } = await getProxies()
      setProxies(data.proxies ?? [])
    } catch {
      message.error(t('proxies.failedToLoad'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchProxies()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  async function handleAdd(values: { host: string; port: string; protocol: string; username?: string; password?: string }) {
    setSubmitting(true)
    try {
      await createProxy(values)
      message.success(t('proxies.added'))
      setModalOpen(false)
      form.resetFields()
      fetchProxies()
    } catch {
      message.error(t('proxies.failedToAdd'))
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteProxy(id)
      message.success(t('proxies.deleted'))
      fetchProxies()
    } catch {
      message.error(t('proxies.failedToDelete'))
    }
  }

  async function handleTest(id: number) {
    setTestingId(id)
    try {
      console.info('Testing proxy', id)
      await testProxy(id)
      console.info('Proxy is reachable')
      message.success(t('proxies.reachable'))
    } catch {
      message.error(t('proxies.testFailed'))
    } finally {
      setTestingId(null)
    }
  }

  const columns: TableProps<Proxy>['columns'] = [
    { title: t('proxies.host'), dataIndex: 'host', key: 'host' },
    { title: t('proxies.port'), dataIndex: 'port', key: 'port' },
    { title: t('proxies.protocol'), dataIndex: 'protocol', key: 'protocol' },
    { title: t('proxies.username'), dataIndex: 'username', key: 'username' },
    {
      title: t('common.status'),
      dataIndex: 'status',
      key: 'status',
      render: (v) => (
        v === 'active'
          ? <Tag color="green">{t('proxies.statusActive')}</Tag>
          : <Tag color="red">{t('proxies.statusInactive')}</Tag>
      ),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Button
            size="small"
            loading={testingId === record.id}
            onClick={() => handleTest(record.id)}
          >
            {t('common.test')}
          </Button>
          <Popconfirm
            title={t('proxies.deleteConfirm')}
            onConfirm={() => handleDelete(record.id)}
            okText={t('common.yes')}
            cancelText={t('common.no')}
          >
            <Button size="small" danger>
              {t('common.delete')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>
          {t('proxies.title')}
        </Title>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            form.resetFields()
            setModalOpen(true)
          }}
        >
          {t('proxies.addProxy')}
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={proxies}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 20 }}
      />

      <Modal
        title={t('proxies.addProxy')}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={submitting}
        okText={t('common.add')}
        cancelText={t('common.cancel')}
      >
        <Form form={form} layout="vertical" onFinish={handleAdd}>
          <Form.Item name="host" label={t('proxies.host')} rules={[{ required: true }]}>
            <Input placeholder={t('proxies.hostPlaceholder')} />
          </Form.Item>
          <Form.Item name="port" label={t('proxies.port')} rules={[{ required: true }]}>
            <Input placeholder={t('proxies.portPlaceholder')} />
          </Form.Item>
          <Form.Item name="protocol" label={t('proxies.protocol')} rules={[{ required: true }]} initialValue="http">
            <Select>
              {PROXY_PROTOCOLS.map((proto) => (
                <Select.Option key={proto} value={proto}>
                  {proto}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="username" label={t('proxies.username')}>
            <Input placeholder={t('common.optional')} />
          </Form.Item>
          <Form.Item name="password" label={t('proxies.password')}>
            <Input.Password placeholder={t('common.optional')} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}


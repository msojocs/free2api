import { useEffect, useState, type Key } from 'react'
import {
  App as AntdApp,
  Table,
  Button,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Steps,
  Space,
  Switch,
  Typography,
  Popconfirm,
  DatePicker,
  type TableProps,
} from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import StatusTag from '../components/StatusTag'
import TaskProgress from '../components/TaskProgress'
import { getTasks, createTask, startTask, pauseTask, deleteTask, type TaskBatch } from '../api/tasks'
import { getTempMailProviders, type TempMailProvider } from '../api/tempMailProviders'
import { getProxyGroups, type ProxyGroup } from '../api/proxyGroups'
import dayjs from 'dayjs'

const { Title } = Typography

const PLATFORMS = [
  { label: 'ChatGPT', value: 'chatgpt' },
  { label: 'Cursor', value: 'cursor' },
]

type WizardValues = {
  type?: string
  total?: number
  proxy_group_id?: number | ''
  temp_mail_provider_id?: number | ''
  mail_use_proxy?: boolean
  concurrency?: number
  interval_seconds?: number
  scheduled_at?: dayjs.Dayjs
}

export default function TaskList() {
  const [tasks, setTasks] = useState<TaskBatch[]>([])
  const [loading, setLoading] = useState(false)
  const [batchDeleting, setBatchDeleting] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([])
  const [wizardOpen, setWizardOpen] = useState(false)
  const [currentStep, setCurrentStep] = useState(0)
  const [wizardValues, setWizardValues] = useState<WizardValues>({})
  const [submitting, setSubmitting] = useState(false)
  const [progressTaskId, setProgressTaskId] = useState<number | null>(null)
  const [progressTaskStatus, setProgressTaskStatus] = useState<TaskBatch['status'] | null>(null)
  const [tempMailProviders, setTempMailProviders] = useState<TempMailProvider[]>([])
  const [proxyGroups, setProxyGroups] = useState<ProxyGroup[]>([])
  const [form] = Form.useForm()
  const { t } = useTranslation()
  const { message } = AntdApp.useApp()

  async function fetchTasks() {
    setLoading(true)
    try {
      const { data } = await getTasks()
      setTasks(data.tasks ?? [])
    } catch {
      message.error(t('tasks.failedToLoad'))
    } finally {
      setLoading(false)
    }
  }

  async function fetchTempMailProviders() {
    try {
      const { data } = await getTempMailProviders()
      setTempMailProviders((data.providers ?? []).filter((p) => p.enabled))
    } catch {
      // non-fatal — provider list will be empty
    }
  }

  async function fetchProxyGroups() {
    try {
      const { data } = await getProxyGroups()
      setProxyGroups(data.groups ?? [])
    } catch {
      setProxyGroups([])
    }
  }

  useEffect(() => {
    fetchTasks()
    fetchTempMailProviders()
    fetchProxyGroups()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (wizardOpen) {
      form.setFieldsValue(wizardValues)
    }
  }, [currentStep, form, wizardOpen, wizardValues])

  function openWizard() {
    const initialValues: WizardValues = {
      concurrency: 5,
      interval_seconds: 5,
      mail_use_proxy: true,
    }
    setCurrentStep(0)
    setWizardValues(initialValues)
    form.setFieldsValue(initialValues)
    setWizardOpen(true)
  }

  async function handleNextStep() {
    const vals = await form.validateFields()
    const merged = { ...wizardValues, ...vals }
    setWizardValues(merged)

    if (currentStep < 2) {
      setCurrentStep((s) => s + 1)
    } else {
      setSubmitting(true)
      try {
        const cfg: Record<string, unknown> = {
          proxy_group_id: merged.proxy_group_id,
          mail_use_proxy: merged.mail_use_proxy !== false,
          concurrency: merged.concurrency ?? 5,
          interval_seconds: merged.interval_seconds ?? 5,
          scheduled_at: merged.scheduled_at?.toISOString(),
        }
        // Include temp mail provider ID if selected so the backend can resolve it.
        if (merged.temp_mail_provider_id) {
          cfg.temp_mail_provider_id = merged.temp_mail_provider_id
        }
        await createTask({
          type: merged.type!,
          total: merged.total!,
          config: cfg,
        })
        message.success(t('tasks.created'))
        setWizardOpen(false)
        fetchTasks()
      } catch {
        message.error(t('tasks.failedToCreate'))
      } finally {
        setSubmitting(false)
      }
    }
  }

  function handlePrevStep() {
    const vals = form.getFieldsValue()
    const merged = { ...wizardValues, ...vals }
    setWizardValues(merged)
    setCurrentStep((s) => Math.max(0, s - 1))
  }

  async function handleStart(id: number) {
    try {
      await startTask(id)
      message.success(t('tasks.started'))
      fetchTasks()
    } catch {
      message.error(t('tasks.failedToStart'))
    }
  }

  async function handlePause(id: number) {
    try {
      await pauseTask(id)
      message.success(t('tasks.paused'))
      fetchTasks()
    } catch {
      message.error(t('tasks.failedToPause'))
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteTask(id)
      message.success(t('tasks.deleted'))
      fetchTasks()
    } catch {
      message.error(t('tasks.failedToDelete'))
    }
  }

  async function handleBatchDelete() {
    if (selectedRowKeys.length === 0) {
      message.warning(t('tasks.batchDeleteSelectFirst'))
      return
    }

    setBatchDeleting(true)
    let successCount = 0
    try {
      const ids = selectedRowKeys.map((key) => Number(key)).filter((id) => Number.isFinite(id))
      for (const id of ids) {
        try {
          await deleteTask(id)
          successCount += 1
        } catch {
          // continue deleting remaining tasks
        }
      }
      const failedCount = ids.length - successCount
      message.info(
        t('tasks.batchDeleteSummary', {
          total: ids.length,
          success: successCount,
          failed: failedCount,
        }),
      )
      setSelectedRowKeys([])
      await fetchTasks()
    } finally {
      setBatchDeleting(false)
    }
  }

  const columns: TableProps<TaskBatch>['columns'] = [
    { title: t('common.id'), dataIndex: 'id', key: 'id' },
    { title: t('common.type'), dataIndex: 'type', key: 'type' },
    {
      title: t('common.status'),
      dataIndex: 'status',
      key: 'status',
      render: (s) => <StatusTag status={s} />,
    },
    { title: t('tasks.total'), dataIndex: 'total', key: 'total' },
    { title: t('tasks.done'), dataIndex: 'completed', key: 'completed' },
    { title: t('tasks.failed'), dataIndex: 'failed', key: 'failed' },
    {
      title: t('common.created'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (v) => new Date(v).toLocaleString(),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      render: (_, record) => (
        <Space>
          {record.status === 'pending' || record.status === 'paused' ? (
            <Button size="small" type="primary" onClick={() => handleStart(record.id)}>
              {t('tasks.start')}
            </Button>
          ) : null}
          {record.status === 'running' ? (
            <Button size="small" onClick={() => handlePause(record.id)}>
              {t('tasks.pause')}
            </Button>
          ) : null}
          <Button
            size="small"
            onClick={() => {
              setProgressTaskId(record.id)
              setProgressTaskStatus(record.status)
            }}
          >
            {t('tasks.progress')}
          </Button>
          <Popconfirm
            title={t('tasks.deleteConfirm')}
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

  const stepItems = [
    { title: t('tasks.stepPlatform') },
    { title: t('tasks.stepResources') },
    { title: t('tasks.stepSettings') },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>
          {t('tasks.title')}
        </Title>
        <Space>
          <Popconfirm
            title={t('tasks.batchDeleteConfirm')}
            onConfirm={() => void handleBatchDelete()}
            okText={t('common.yes')}
            cancelText={t('common.no')}
          >
            <Button danger disabled={selectedRowKeys.length === 0} loading={batchDeleting}>
              {t('tasks.batchDelete')}
            </Button>
          </Popconfirm>
          <Button icon={<ReloadOutlined />} onClick={fetchTasks} loading={loading}>
            {t('tasks.refreshList')}
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openWizard}>
            {t('tasks.createTask')}
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={tasks}
        rowKey="id"
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys),
        }}
        loading={loading}
        pagination={{ pageSize: 10 }}
        scroll={{ x: 'max-content' }}
      />

      {/* Create Task Wizard */}
      <Modal
        title={t('tasks.createTask')}
        open={wizardOpen}
        onCancel={() => setWizardOpen(false)}
        footer={
          <Space>
            {currentStep > 0 && (
              <Button onClick={handlePrevStep}>{t('common.back')}</Button>
            )}
            <Button onClick={() => setWizardOpen(false)}>{t('common.cancel')}</Button>
            <Button
              type="primary"
              onClick={handleNextStep}
              loading={submitting}
            >
              {currentStep < 2 ? t('common.next') : t('common.create')}
            </Button>
          </Space>
        }
        width={560}
      >
        <Steps current={currentStep} items={stepItems} style={{ marginBottom: 24 }} />
        <Form form={form} layout="vertical">
          {currentStep === 0 && (
            <>
              <Form.Item
                name="type"
                label={t('tasks.platform')}
                rules={[{ required: true }]}
              >
                <Select placeholder={t('tasks.selectPlatform')} options={PLATFORMS} />
              </Form.Item>
              <Form.Item
                name="total"
                label={t('tasks.accountCount')}
                rules={[{ required: true }]}
              >
                <InputNumber min={1} max={10000} style={{ width: '100%' }} />
              </Form.Item>
            </>
          )}
          {currentStep === 1 && (
            <>
              <Form.Item name="temp_mail_provider_id" label={t('tasks.tempMailProvider')}>
                <Select
                  placeholder={t('tasks.selectTempMailProvider')}
                  allowClear
                  options={[
                    { value: '', label: t('tasks.noTempMailProvider') },
                    ...tempMailProviders.map((p) => ({ value: p.id, label: p.name })),
                  ]}
                />
              </Form.Item>
              <Form.Item name="proxy_group_id" label={t('tasks.proxyGroup')}>
                <Select
                  allowClear
                  placeholder={t('tasks.proxyGroupPlaceholder')}
                  options={proxyGroups.map((group) => ({ value: group.id, label: group.name }))}
                />
              </Form.Item>
              <Form.Item name="mail_use_proxy" label={t('tasks.mailUseProxy')} valuePropName="checked">
                <Switch />
              </Form.Item>
            </>
          )}
          {currentStep === 2 && (
            <>
              <Form.Item
                name="concurrency"
                label={t('tasks.concurrency')}
                rules={[{ required: true }]}
                initialValue={5}
              >
                <InputNumber min={1} max={100} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item
                name="interval_seconds"
                label={t('tasks.intervalSeconds')}
                rules={[{ required: true }]}
                initialValue={5}
              >
                <InputNumber min={0} max={3600} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item name="scheduled_at" label={t('tasks.scheduledTime')}>
                <DatePicker showTime style={{ width: '100%' }} />
              </Form.Item>
            </>
          )}
        </Form>
      </Modal>

      {/* Task Progress Modal */}
      <TaskProgress
        taskId={progressTaskId}
        taskStatus={progressTaskStatus}
        open={progressTaskId != null}
        onClose={() => {
          setProgressTaskId(null)
          setProgressTaskStatus(null)
        }}
      />
    </div>
  )
}

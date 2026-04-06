import { useEffect, useState } from 'react'
import { App as AntdApp, Button, Form, Input, Modal, Popconfirm, Space, Table, Typography, type TableProps } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { createProxyGroup, deleteProxyGroup, getProxyGroups, updateProxyGroup, type ProxyGroup } from '../api/proxyGroups'

const { Title } = Typography

export default function ProxyGroupManager() {
  const [groups, setGroups] = useState<ProxyGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingGroup, setEditingGroup] = useState<ProxyGroup | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm<{ name: string }>()
  const { t } = useTranslation()
  const { message } = AntdApp.useApp()

  async function fetchGroups() {
    setLoading(true)
    try {
      const { data } = await getProxyGroups()
      setGroups(data.groups ?? [])
    } catch {
      message.error(t('proxyGroups.failedToLoad'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchGroups()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function openCreate() {
    setEditingGroup(null)
    form.setFieldsValue({ name: '' })
    setModalOpen(true)
  }

  function openEdit(group: ProxyGroup) {
    setEditingGroup(group)
    form.setFieldsValue({ name: group.name })
    setModalOpen(true)
  }

  function closeModal() {
    setModalOpen(false)
    setEditingGroup(null)
    form.resetFields()
  }

  async function handleSubmit(values: { name: string }) {
    setSubmitting(true)
    try {
      if (editingGroup) {
        await updateProxyGroup(editingGroup.id, { name: values.name.trim() })
        message.success(t('proxyGroups.updated'))
      } else {
        await createProxyGroup({ name: values.name.trim() })
        message.success(t('proxyGroups.created'))
      }
      closeModal()
      await fetchGroups()
    } catch (error) {
      const fallback = editingGroup ? t('proxyGroups.failedToUpdate') : t('proxyGroups.failedToCreate')
      if (error instanceof Error && error.message) {
        message.error(error.message)
      } else {
        message.error(fallback)
      }
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteProxyGroup(id)
      message.success(t('proxyGroups.deleted'))
      await fetchGroups()
    } catch (error) {
      if (error instanceof Error && error.message) {
        message.error(error.message)
      } else {
        message.error(t('proxyGroups.failedToDelete'))
      }
    }
  }

  const columns: TableProps<ProxyGroup>['columns'] = [
    { title: t('common.name'), dataIndex: 'name', key: 'name' },
    {
      title: t('common.created'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (value: string) => new Date(value).toLocaleString(),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Button size="small" onClick={() => openEdit(record)}>
            {t('common.edit')}
          </Button>
          <Popconfirm
            title={t('proxyGroups.deleteConfirm')}
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
          {t('proxyGroups.title')}
        </Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          {t('proxyGroups.addGroup')}
        </Button>
      </div>

      <Table columns={columns} dataSource={groups} rowKey="id" loading={loading} pagination={false} />

      <Modal
        title={editingGroup ? t('proxyGroups.editTitle', { name: editingGroup.name }) : t('proxyGroups.newTitle')}
        open={modalOpen}
        onCancel={closeModal}
        onOk={() => form.submit()}
        confirmLoading={submitting}
        okText={editingGroup ? t('common.save') : t('common.add')}
        cancelText={t('common.cancel')}
        destroyOnHidden
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label={t('common.name')} rules={[{ required: true }]}>
            <Input placeholder={t('proxyGroups.namePlaceholder')} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
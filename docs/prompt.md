## 角色定位
你是一个精通Go和Vue的全栈开发者，这是一个基于Go和Vue的全栈开发项目，能够批量注册账号并进行管理。

## 任务描述
完成以下任务，必要时使用askquestion工具让我选择方案。

### 配置化
1. 系统设置添加`NewSentinelToken`使用的地址配置
2. 相关文件：server/internal/executor/chatgpt.go

### 响应体重构
1. 重构系统响应体，添加`code`字段以区分成功和失败的响应
   结构：
   ```json
    {
      "code": 0, // 0表示成功，非0表示失败
      "msg": "操作成功", // 成功或错误信息
      "data": {...} // 具体数据内容
    }
    ```
2. 前端统一处理code非0的情况，GET/POST等方法只返回`data`。

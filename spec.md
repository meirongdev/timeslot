# Timeslot 系统设计方案 (Privacy-First)

## 核心设计理念
1. **隐私优先**：对外仅暴露“空闲/占用”状态，严禁泄露具体日程内容。
2. **极简集成**：通过 Hugo 博客展示状态，仅用于展示可用时间段，降低攻击面。
3. **HomeLab 友好**：支持 Zitadel (SSO)、Gotify (推送)、OpenTelemetry (监控)。

## 架构概览
┌─────────────────────────────────────────────────────┐
│                    你的 Homelab                       │
│                                                       │
│  ┌─────────────┐    ┌──────────────┐    ┌─────────┐ │
│  │ Calendar    │    │   Core       │    │  Admin  │ │
│  │ Sync Worker │───►│   Engine     │◄───│   UI    │ │
│  └─────────────┘    └──────┬───────┘    └─────────┘ │
│                             │             (Zitadel) │
│                      ┌──────▼───────┐                │
│                      │   SQLite     │                │
│                      └──────┬───────┘                │
│                             │             (OTel)    │
│                      ┌──────▼───────┐                │
│                      │  Privacy API │                │
│                      └──────┬───────┘                │
└─────────────────────────────┼───────────────────────┘
                               │
                    ┌──────────▼──────────┐
                    │    Hugo 博客 (Read)  │
                    │      (JS Widget)     │
                    └─────────────────────┘

## 核心模块
1. **Calendar Sync Worker**: 定时拉取 iCal 并更新 `busy_blocks`。
2. **Core Engine**: 根据 `availability_rules` 减去 `busy_blocks` 计算状态。
3. **Privacy API**:
   - `GET /api/slots`: 返回可用的时间段。
   - `GET /api/schedule`: 返回脱敏后的时间轴（Available/Occupied）。
4. **Admin UI**:
   - 必须通过 Zitadel (OIDC) 或 Basic Auth 登录。
   - 管理日历源、设置可用性规则。
5. **Notification**:
   - 通过 Gotify 向管理员推送系统事件（如同步失败）。

## 技术栈
- **Go 1.25**: 单二进制部署，低内存占用。
- **SQLite**: 简单、可靠的本地数据存储。
- **Zitadel**: OIDC 认证集成。
- **Gotify**: 消息推送。
- **OpenTelemetry**: 链路追踪与指标。
- **Hugo**: 前端展示，通过原生 JS 调用脱敏 API。

## 数据脱敏逻辑
- **对外 (Public)**: `Slot{ Start, End, Status: "available" | "occupied" }`
- **对内 (Admin)**: `Event{ Start, End, Title, SourceCalendar }`

# TexasPoker (Web Multiplayer)

这个项目实现了网页版多人在线德州扑克：
- Go 后端：SSE 实时通信、房间/玩家管理、牌局状态机
- React 前端：兼容桌面/手机，黑白涂鸦风牌桌 UI

## 你提出的关键问题：这是不是后端逻辑？
是。网络通信、牌局规则、玩家管理都属于后端核心逻辑；前端主要负责渲染状态和发送玩家操作。

## 参考来源说明
你给的仓库 `ratel-online/server` 在当前环境无法直接读取 GitHub 源码页面，我基于其在 `pkg.go.dev` 可见的包结构进行复现，核心思路映射如下：
- `network` -> `server/internal/network`（实时连接与消息推送）
- `state / center` -> `server/internal/room`（房间与玩家会话管理）
- `game/texas + rule` -> `server/internal/engine`（发牌、盲注、行动轮、showdown 结算）

## 目录

- `server/cmd/pokerd/main.go`：服务入口
- `server/internal/network/sse.go`：SSE 连接与 action API
- `server/internal/room/room.go`：房间与消息路由
- `server/internal/engine/*`：牌局引擎与牌型比较
- `web/src/App.tsx`：牌桌主界面
- `web/src/styles.css`：黑白涂鸦风响应式样式

## 后端启动

```bash
cd server
go run ./cmd/pokerd
```

默认监听 `:8080`，健康检查：`/healthz`

## 前端启动

```bash
cd web
npm install
npm run dev
```

默认地址：`http://localhost:5173`

API 默认连 `http://<当前主机>:8080`，可用环境变量覆盖：

```bash
VITE_API_URL=http://localhost:8080 npm run dev
```

## 连接方式

前端 URL 可带参数：
- `room`：房间号
- `user`：用户 id
- `name`：展示名

示例：

```
http://localhost:5173/?room=alpha&user=u1&name=Alice
http://localhost:5173/?room=alpha&user=u2&name=Bob
```

## 协议

客户端 -> 服务端（POST `/action?room=...&user=...`）：
- `{"type":"start_hand","payload":{}}`
- `{"type":"action","payload":{"action":"fold|check|call|bet|raise|all_in","amount":40}}`

服务端 -> 客户端（GET `/events?room=...&user=...&name=...`，SSE 推流）：
- `{"type":"snapshot","payload":{...牌局全量状态...}}`
- `{"type":"error","payload":{"message":"..."}}`

## Git 使用

```bash
git init
git add .
git commit -m "feat: bootstrap multiplayer texas poker backend and react table ui"
```

## 让其他人加入（公网部署）

当前项目默认是本地开发模式；如果要让其他人加入，需要把后端部署到一台可公网访问的服务器。

最小步骤：

1. 准备一台云服务器（Linux，开放 80/443 端口）。
2. 在服务器启动后端：
   ```bash
   cd server
   go run ./cmd/pokerd
   ```
3. 用 Nginx/Caddy 反向代理到 `localhost:8080`，并配置 HTTPS。
4. 前端构建时把 API 地址指向公网域名：
   ```bash
   cd web
   VITE_API_URL=https://your-domain.com npm run build
   ```
5. 把 `web/dist` 部署到静态站点服务（Nginx/Vercel/Netlify 均可）。

玩家只要访问你的前端地址，并使用同一个 `room` 参数，就能进入同一张桌子。

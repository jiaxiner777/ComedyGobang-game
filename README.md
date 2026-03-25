# Code Breaker: The Matrix Roguelike

本项目已重构为前后端分离的本地网页游戏架构：

- `cmd/codebreaker-web/`：程序入口，负责组装 HTTP 服务
- `internal/api/`：后端接口层，暴露会话、移动、战斗、奖励等 API
- `internal/game/`：核心游戏引擎，包含地图、战斗、卡牌、奖励与状态流转
- `frontend/`：前端静态资源与嵌入逻辑

## 运行开发版

```bash
go run ./cmd/codebreaker-web
```

默认访问地址： [http://localhost:8080](http://localhost:8080)

也可以通过环境变量修改端口：

```bash
CODEBREAKER_ADDR=:9000 go run ./cmd/codebreaker-web
```

## 构建

```bash
go build -o dist/codebreaker_web_refactor.exe ./cmd/codebreaker-web
```

## 当前产物

- 根目录 `codebreaker_web.exe`：旧的可玩版本，已保留
- `dist/codebreaker_web_refactor.exe`：当前重构后的新版本

## 当前实现

- 本地前端 + Go 后端服务，无外部依赖
- 浏览器可玩的树形地图探索、战斗、宝物、奖励、删牌与结局流程
- 以算法卡牌驱动的战斗系统，敌人 bits 会实时映射为生命、攻击、防御、倒计时等状态
- 支持遍历协议、同步奖励、失衡事件、遗物、剧情碎片与负载系统
- 可继续打包为桌面软件或迭代接入更完整的前端工程体系

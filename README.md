# MiniPrometheus

参考 Prometheus 的自研 TSDB 内核：Delta-of-Delta + Gorilla XOR、手写倒排位图、MiniQL 执行树、双分片采集大屏。

## 1. 如何启动

```bash
docker compose up --build -d
```

浏览器打开 `http://localhost:31871`。无需手工装 Go / Node。

## 2. 使用说明

左侧三字导航：**流** 看 Canvas 曲线，**析** 输入 MiniQL 并查看后端执行树，**群** 看分片健康、压缩率与标签位图。时间均为北京时间 `yyyy-MM-dd HH:mm:ss`。顶部琥珀条表示分片降级，结果可能残缺。

## 3. 服务列表及API说明

| 入口 | 说明 |
|---|---|
| http://localhost:31871 | 大屏（Nginx 反代 /api /health /ws） |
| http://localhost:31872 | Gateway 直连 |
| http://localhost:31873 / 31874 | Storage 分片 |
| http://localhost:31875 | Agent |

完整契约见 `docs/API.md`。

## 4. 测试账号

无登录。观测台只读。

## 5. 题目内容

工业级海量监控场景下的高频多维时序写入、极致压缩与倒排检索，以及 PromQL 子集的可视化执行剖析。

## 6. 项目结构

```
backend/           Go TSDB / MiniQL / Agent / Gateway
frontend-user/     Vue 3 大屏
docs/              需求与设计
tests/             Playwright + API smoke
docker-compose.yml
```

## 7. API 模拟与切换指南

真实路径：Agent 抓取 compose 内 `node-exporter` 与各组件 `/metrics`（Prometheus 文本协议）。

模拟路径：内置合成负载发生器，用于制造真实 exporter 给不出的高频压力。

切换：环境变量 `MP_SCRAPE_MODE=real|synthetic|both`，默认 `both`。改 `docker-compose.yml` 中 agent 服务后 `docker compose up -d agent`。

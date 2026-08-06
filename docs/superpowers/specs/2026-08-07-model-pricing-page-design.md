# 模型定价页重建设计

## 目标

将 `/app/available-channels` 从“按渠道展示支持模型”改为“按模型展示定价”，支持 Anthropic/OpenAI 标签切换、搜索、排序、供应商标识、分组倍率和模型价格。

## 数据方案

生产探查结果：

- `GET /api/v1/models` 返回 404。
- `GET /api/v1/pricing` 返回 404。
- `GET /api/v1/groups/available` 未带登录凭据时返回 401。
- 项目已有的认证用户接口 `/api/v1/channels/available` 返回 `channels[] -> platforms[] -> supported_models[]`，模型对象包含 `pricing.input_price`、`pricing.output_price`、`pricing.cache_write_price`、`pricing.cache_read_price`、`pricing.intervals` 等字段。

因此不新增独立 pricing API；页面复用现有 `getAvailable()` 和 `getUserGroupRates()`，避免重复请求和新增无效接口。

## 页面结构

新建 `frontend/src/views/user/ModelPricingView.vue`：

1. 使用现有 `AppSectionShell`，标题沿用 `availableChannels` 文案。
2. 顶部提供 Anthropic/OpenAI 两个切换标签，按模型平台过滤。
3. 提供模型搜索框和刷新按钮；显示当前过滤后的数量。
4. 表格列为：模型名、供应商、分组、输入、输出、缓存、上下文。
5. 分组列显示第一组作为默认 badge；多组通过下拉展示名称、倍率和描述。
6. 价格显示按美元每百万 token；基准价存在时乘分组倍率，缺失价格显示 `-`。
7. 上下文从现有模型元数据中读取；接口没有该字段时显示 `-`，不伪造数值。

## 平台归类与图标

优先使用接口返回的 `platform`，再用模型名前缀做兼容兜底：

- `claude-` 归 Anthropic。
- `gpt-`、`o1`、`o3`、`deepseek-` 归 OpenAI。

供应商图标复用 `PlatformIcon`；分组 badge 复用 `GroupBadge`，不复制图标 SVG 或颜色逻辑。

## 路由与发布

将路由 `/app/available-channels` 的懒加载组件改为 `ModelPricingView.vue`，保留 `AvailableChannelsView.vue`。构建后只把新页面 chunk、其新增样式/依赖和入口中的对应路由映射加入隔离发布目录，不替换无关主包资源。

## 验证

- `npm run type-check`
- `npm run build`
- 针对平台归类、价格换算、搜索和排序增加/运行单元测试。
- 检查生产新 chunk HTTP 200、入口映射正确、服务 health 200。
- 生产页面验收：两个标签可切换，Claude/GPT 归类正确，图标颜色正确，分组倍率和搜索生效。

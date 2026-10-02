export type GuideTool = 'codex' | 'claude' | 'image'
export type GuideSystem = 'windows' | 'mac'

export function installationGuide(tool: GuideTool, system: GuideSystem): string {
  const windows = system === 'windows'
  const installSwitch = windows
    ? '选择 **Windows**。普通 Intel / AMD 电脑选 **x64 的 .msi 安装包**；只有系统信息明确写 ARM64 时才选 ARM64。下载后打开安装包，按提示完成安装，再从开始菜单打开 CC Switch。'
    : '选择 **macOS 的 .dmg 安装包**。下载后双击，将 CC Switch 拖进 Applications（应用程序），再从“应用程序”打开它。'
  const client = tool === 'codex'
    ? `## 第二步：下载并安装 Codex

[↓ 下载 Codex · 官方入口 ↗](https://developers.openai.com/codex/app/)

点击上方蓝色按钮，在官方页面选择 **${windows ? 'Windows' : 'macOS'}** 下载并安装。官方新版可能显示为 ChatGPT 桌面版，请进入其中的 Codex 本地工作功能，而不是普通网页聊天。

先安装好即可，**先不要发送任务**。下面完成 SSXZ 导入和启用后，再完全退出、重新打开客户端。CC Switch 的 Codex 配置与 Claude 配置是分开的。

如果登录页要求登录，使用 **Sign in another way（其他方式登录）/ API Key**；不要把 SSXZ 的邮箱密码当作 OpenAI 账号密码。先完成第三、第四步的 SSXZ 配置，再使用本站密钥。`
    : `## 第二步：安装 Claude Code

这里教的是 **Claude Code**，不是 Claude 网页聊天，也不是 Claude Desktop。

${windows ? '点击 Windows 开始菜单，输入 **PowerShell** 并打开。' : '按 Command + 空格，输入 **终端** 并打开。'}下面这一行是 Anthropic 官方安装命令：复制整行，粘贴到刚打开的窗口，按回车。它会下载并安装 Claude Code。

\`\`\`${windows ? 'powershell' : 'bash'}
${windows ? 'irm https://claude.ai/install.ps1 | iex' : 'curl -fsSL https://claude.ai/install.sh | bash'}
\`\`\`

完成后关闭这个窗口，重新打开${windows ? ' PowerShell' : '终端'}，输入 \`claude --version\` 并回车。**看到版本号才算安装成功**。暂时不发送任务，先完成下面的 SSXZ 配置。

[查看 Claude Code 官方安装说明](https://code.claude.com/docs/en/setup)`
  return `## 第一步：下载并安装 CC Switch

**要装两个东西：CC Switch 帮你配置，${tool === 'codex' ? 'Codex' : 'Claude Code'} 帮你做事。装了前者不等于装了后者。** 已装好的部分可以跳过。

[↓ 下载 CC Switch · 官方入口 ↗](https://ccswitch.io/zh/download)

1. 点击上方蓝色按钮打开官方下载页。**不需要注册 GitHub，也不需要购买 CC Switch。**
2. ${installSwitch}
3. 打开后能看到供应商列表，才继续下面的步骤。

![官方下载页：先选系统，再选安装版；普通 Windows 电脑点 x64（截图来源 ccswitch.io）](./assets/cc-switch/08-official-direct-download.jpg)

![CC Switch 主界面示例；下步要切换到你选择的软件](./assets/cc-switch/02-cc-switch-main-window.png)

${client}

> 安装遇到安全警告或下载失败，先核对来源，不要关闭杀毒软件或执行来路不明的“修复命令”。备用下载在 [官方 GitHub 发布页](https://github.com/farion1231/cc-switch/releases/latest)，往下找到 Assets（安装包），选对应系统文件，不选 Source code（源码）。`
}

export function connectionGuide(tool: GuideTool): string {
  const codex = tool === 'codex'
  const client = codex ? 'Codex' : 'Claude Code'
  return `## 第三步：在本站创建 API 密钥

API Key（密钥）就是让软件使用你 SSXZ 额度的钥匙，**不是登录密码，不要发给别人**。

1. [打开本站 API 密钥页面](/app/keys)。没登录会先要求登录；没有本站账号，先完成注册再回到此页。
2. 已经有适合的密钥就复用，不需要重复创建。列表为空时，点 **创建密钥**。
3. 名称填“我的 ${client}”。分组选 **${codex ? '支持 GPT 的分组' : '支持 Claude 的分组'}**，不要选图片专用分组。到 [模型页面](/app/available-channels) 确认想用的模型在该分组里。
4. 按页面提示创建。回到列表，确认密钥是启用的、未过期，本站有可用额度。密钥若单独设置了额度上限，也不能已用完。

> 分组可以理解成“套餐线路”，决定能用什么模型和价格。模型与价格以你自己的模型页面为准，不是选了便宜分组就能用全部模型。

![在 API 密钥列表找到“导入到 CCS”；截图是按钮位置示例](./assets/cc-switch/03-ssxz-import-to-ccs-button.png)

## 第四步：导入 SSXZ 并启用

1. 在刚才那把密钥所在行，点击 **导入到 CCS**。
2. 浏览器询问是否打开 CC Switch 时，点击 **打开 CC Switch**。这一步把配置交给你电脑上的 CC Switch。

![点击“打开 CC Switch”允许浏览器唤起已安装的软件](./assets/cc-switch/04-browser-open-cc-switch-dialog.png)

3. 核对导入目标是 **${client}**，名称是 SSXZ。软件目标由密钥所属分组决定；如果导入到了另一种软件，先回本站检查分组，不要硬套。
4. 地址应该是 **\`${codex ? 'https://api.ssxzapi.com/v1' : 'https://api.ssxzapi.com'}\`**。${codex ? 'Codex 的地址带 /v1。' : 'Claude Code 的地址不要额外加 /v1。'}密钥应已带入，模型应属于刚选的分组。
5. 点击 **导入**。回到 CC Switch 的 **${client}** 页，找到 SSXZ，点击 **启用**（已显示使用中则不用再点）。

${codex ? '> 不要照着 Claude 的截图选择 Claude 页签；你这里要选 Codex。' : '![在 Claude 页确认 SSXZ 已启用；截图中的模型不代表所有分组都支持](./assets/cc-switch/06-ssxz-selected-redacted.png)'}

> 更换已有供应商会改变客户端之后使用的线路。需要保留旧配置时，先用 CC Switch 的备份功能备份；不要删除整个配置目录。

## 第五步：发送消息，确认连接成功

1. 完全退出并重新打开 **${client}**。${codex ? '选择本地任务，选一个不含隐私文件的空文件夹作为测试位置。' : '打开一个空的测试文件夹，在终端进入该文件夹后输入 `claude` 并回车。只信任你认识的文件夹。'}
2. 输入：**只回复：连接成功。不要读取或修改文件。** 然后发送。这会产生少量实际用量，按你选择的分组计费。
3. 收到回复后，回到本站 [使用记录](/app/usage)，检查刚才时间附近是否出现对应密钥和模型的记录。**有回复且本站有对应记录，才确认走的是 SSXZ。**

${codex ? '' : '![Claude Code 回复成功的历史界面示例；你应以刚才自己的回复为准](./assets/cc-switch/07-claude-code-success-history.png)'}

没有成功时，先打开下面与你遇到的现象对应的帮助，不要反复创建密钥或充值。`
}

export const imageGuide = `图片工作台直接在浏览器中使用，**不用安装 CC Switch 或其他软件**。

## 第一步：打开图片工作台

[打开图片工作台 →](/app/image)

先登录本站，再点击上方入口，或从左侧菜单进入 **图片工作台**。不要把密钥拼进网址发给别人。

## 第二步：选择图片来源和模型

若工作台要求配置来源，选 SSXZ 对应来源，并使用具备图片分组权限的本站密钥。文字用的 GPT 分组不一定能画图。选择你有权限的图片模型，先用默认尺寸、**1 张图片**测试。

## 第三步：填写要求，点击生成

第一次可以输入：**一只戴蓝色围巾的橘猫，纯白背景，无文字。**

需要按参考图修改时，先上传参考图，再说明保留什么、改变什么。点击生成一次，等待结果，不要连续点击。

## 第四步：查看图片和费用

看到图片后再下载；回本站 [使用记录](/app/usage) 检查实际费用。**看到图片并能正常打开下载结果，才算这次生成完成。**

> 图片生成会实际扣费。出现“Failed to fetch”或等待很久时，先查看任务记录和用量，确认是否还在处理，**不要连续重试**。没有显示图片不等于上游没有处理，也不能单凭错误提示判断是否应扣费。

### 提示词怎样写清楚？

按“主体 + 构图 + 风格 + 必须保留 + 不要出现”描述。需要多视图时写清楚要几张或一张拼图，以及每个视角；如果要求“全身不露脸、头部特写露脸”，分两次生成更容易检查。生成结果不完全遵从时，先核对任务详情中的输入是否与原文一致，再判断模型效果。

### 失败时给客服什么？

提供任务时间、模型、任务编号（如有）、错误截图和是否产生扣费。**不要提供完整密钥**，也不要公开包含私人参考图的任务。`

# Wave 1 第 2 批颜色采样

采样来源：本批 Playwright 截图，1440px 桌面视口；RGB 取截图像素，`B-R` 为蓝通道减红通道。

| 文件 | 采样点 | RGB | B-R | 说明 |
| --- | --- | --- | ---: | --- |
| `dashboard-desktop-dark.png` | `(0,0)` | `27,27,27` | `0` | 深色页面/侧栏基底 |
| `dashboard-desktop-dark.png` | `(720,1310)` | `26,26,26` | `0` | 深色内容面 |
| `dashboard-desktop-light.png` | `(0,0)` | `255,255,255` | `0` | 亮色页面基底 |
| `tour-first-dark.png` | `(20,100)` | `7,8,9` | `2` | 遮罩叠加侧栏 |
| `tour-first-light.png` | `(20,100)` | `3,5,6` | `3` | 遮罩叠加侧栏 |
| `keys-create-dialog-dark.png` | `(720,724)` | `27,27,27` | `0` | 弹窗面板 |
| `keys-create-dialog-light.png` | `(720,724)` | `255,255,255` | `0` | 弹窗面板 |

结论：深色页面、卡片、弹窗和 tour 遮罩均满足 `B-R <= 3`；钢蓝只出现在主操作/激活/焦点等点缀位置。

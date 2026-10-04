[English](README.md) | [Русский](README.ru.md) | 简体中文

# Just Monitor Control

适用于 Windows 的 Stream Deck 插件。可控制显示器亮度、视频输入、刷新率、HDR 和睡眠。

## 功能

| 操作 | 说明 |
| --- | --- |
| 亮度 | 可以设定精确亮度、按步进增减，或在两个已保存的状态之间切换，同时用于多台显示器。 |
| 切换输入 | 切换到指定端口，或在两个端口之间切换，例如 DisplayPort 和 HDMI。 |
| 更改刷新率 | Windows 报告的刷新率，包括 59.94 Hz 这类精确模式。 |
| HDR | 为单台或所有所选显示器开启或关闭 HDR。 |
| 睡眠与唤醒 | 让所选显示器或整个桌面进入睡眠，然后再唤醒。 |
| Raw VCP | 任意 VESA VCP 代码和数值，用于没有单独按钮的控制项。 |
| 识别显示器 | 每块屏幕上会短暂显示编号，以便把下拉菜单中的名称对应到真实显示器。 |

## 要求

- **Elgato Stream Deck 软件**：6.9 或更高版本。
- **系统**：Windows 10（1809 或更高版本）或 Windows 11。
- 在显示器菜单中启用 DDC/CI。
- 通过 DisplayPort 或 HDMI 直接连接。部分 USB-C 集线器和无源转接器不会把指令传到显示器。

## 安装

1. 从 [Releases](https://github.com/valentderah/stream-deck-just-monitor-control/releases) 页面下载最新版本。
2. 双击已下载的 `.streamDeckPlugin` 文件以完成安装。

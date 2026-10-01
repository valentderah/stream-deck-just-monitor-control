English | [Русский](README.ru.md)

# Just Monitor Control

A Stream Deck plugin that controls monitors on Windows: brightness, video input, refresh rate, HDR, and sleep.

## Features

| Action | Description |
| --- | --- |
| Brightness | Set an exact level, step it up or down, or toggle between two saved levels. Several monitors can be changed at once. |
| Input Switch | Switch a monitor to a chosen port, or toggle between two ports such as DisplayPort and HDMI. In toggle mode the button changes only after the monitor confirms the new input. |
| Refresh Rate | Switch between the refresh rates reported by Windows, including exact modes such as 59.94 Hz. |
| HDR | Turn HDR on or off for one monitor or for every selected monitor. |
| Sleep & Wake | Put selected monitors to sleep, or sleep the whole desktop, and wake them again. |
| Raw VCP | Send any VESA VCP code and value for controls that do not have their own button. |
| Identify Displays | Show a temporary number on each screen so the buttons can be matched to the physical monitors. |

## Requirements

- **Elgato Stream Deck Software**: Version 6.9 or higher.
- **Operating System**: Windows 10 (1809+) or Windows 11.
- DDC/CI enabled in the monitor menu.
- A direct DisplayPort or HDMI connection. Some USB-C hubs and passive adapters block monitor control.

## Installation

1. Download the latest version from the [Releases](https://github.com/valentderah/stream-deck-just-monitor-control/releases) page.
2. Double-click the downloaded `.streamDeckPlugin` file to install it.

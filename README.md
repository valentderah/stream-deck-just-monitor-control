# Just Monitor Control

Stream Deck plugin for Windows display management via DDC/CI and native Windows Display APIs. Built with Go (no CGO, ephemeral WinAPI handles), providing low-latency control without relying on abandoned third-party libraries.

## Features

- **Brightness**: Set absolute percentage, step up/down by configurable delta, or toggle between two defined levels (Day/Night). Supports parallel multi-monitor updates.
- **Input Switch**: Switch video inputs directly or toggle between two designated ports (e.g., DisplayPort and HDMI) with dual-state button tracking.
- **Refresh Rate**: Change display refresh rates using Windows-enumerated rational modes (e.g., exact 59.94 Hz or 143.85 Hz) to prevent black screens.
- **HDR**: Toggle Windows Advanced Color (HDR) per monitor or across all selected displays.
- **Sleep & Wake**: Put selected displays into standby via DDC/CI or sleep the entire desktop session via DPMS, with reliable wake-up via synthetic input nudge.
- **Raw VCP**: Send arbitrary VESA MCCS VCP feature codes and values for advanced hardware automation.
- **Identify Displays**: Show temporary numbered overlays on each screen directly from the Property Inspector to match physical monitors with system display IDs.

## Requirements

- Windows 10 (1809+) or Windows 11.
- DDC/CI enabled in the monitor's built-in OSD menu.
- Direct DisplayPort or HDMI connection (some USB-C hubs and passive adapters block DDC/CI I2C communication).

## Installation

1. Download the latest `com.valentderah.just-monitor-control.streamDeckPlugin` from the [Releases](../../releases) section.
2. Double-click the downloaded file to install it into the Elgato Stream Deck software.
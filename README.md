# govterm
Simple VT-compatible Linux Terminal Emulator

[![Licence](https://img.shields.io/badge/Licence-Apache-brightgreen)](https://github.com/kinabcd/govterm/blob/main/LICENSE)
[![Golang](https://img.shields.io/badge/go-1.18+-blue)](https://go.dev/dl/)

------------------------------
[中文](README_zh.md)

## Introduction
**`govterm`** is a pyte-like Linux terminal emulator that simulates a terminal in memory and can be used to parse and process terminal output without the need for an actual physical terminal.

Fork from [go-asiterm](https://github.com/veops/go-ansiterm)

## Core functions

- **`Screen simulation`** includes a screen emulator that can handle the character stream on the screen, supporting operations such as cursor movement and text scrolling.

- **`ANSI control code interpretation`** handles ANSI control code sequence escapes.

## Download
```shell
go get github.com/kinabcd/govterm
```

## Use
```shell
# Create a virtual screen
screen := govterm.NewScreen(80, 24)

# Create a character stream
stream := govterm.NewStream(screen)

# Input characters
stream.WriteString(input)

# Get screen output
output := screen.Display()
```
For more usage examples, see example
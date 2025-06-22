# govterm
Simple VT-compatible Linux Terminal Emulator

[![Licence](https://img.shields.io/badge/Licence-Apache-brightgreen)](https://github.com/kinabcd/govterm/blob/main/LICENSE)
[![Golang](https://img.shields.io/badge/go-1.18+-blue)](https://go.dev/dl/)

------------------------------
[English](README.md)

## 介紹
**`govterm`** 是一款類似pyte的linux终端模擬器，在記憶體中模擬終端，可以用來解析和處理終端的輸出，而不需要實際的物理終端。

起源於 [go-asiterm](https://github.com/veops/go-ansiterm)


## 核心功能

- **`螢幕模擬`** 包含一個螢幕模擬器，可以處理螢幕上的字元流，支援遊標移動、文字滾動等操作。
- **`ANSI 控制碼解釋`** 處理 ANSI 控制碼序列轉義。



## 下載
```shell
go get github.com/kinabcd/govterm
```

## 使用
```shell
# 建立一個虛擬螢幕
screen := govterm.NewScreen(80, 24)

# 建立字元流
stream := govterm.NewStream(screen)

# 輸入字元
stream.WriteString(input)

# 取得螢幕輸出
output := screen.Display()
```
更多使用範例請見 example
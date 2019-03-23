# Jstago
j-stage(https://www.jstage.jst.go.jp)などから論文PDFを探し出して、自動ダウンロードするプログラムです。

## インストールと使用方法
    go get https://github.com/paka3m/jstago
    jstago

## 簡単な説明
適当な論文リンク集のURLを入˝力し、ウィザードに沿ってページ内の走査とダウンロードが始まります。

ダウンロード中もキーボード入力受付します。
- auto[a]: 何か入力するまで、自動ダウンロードが進みます。
- skip[s]: 現在のダウンロードをスキップ
- exit[e], Ctrl+C: プログラム終了

サーバー負荷を避けるために3-10秒のランダム秒待機を実行し、同時ダウンロードは2に制限しています。(仮)

robots.txtで禁止されている場合は、DLを続行するか判断を仰ぎます。自己責任で実行してください。
外部リンクのDLはしません。

## 注意
サーバー攻撃が目的ではありません。
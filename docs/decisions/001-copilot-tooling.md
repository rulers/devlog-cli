# Copilot実装・ツール利用の検証

2026-10-11、GitHub Copilot CLI 1.0.91で `add` の対話入力・本文生成・プレビュー・キャンセルを実装した。投稿処理は含まない。秘密情報を含まない追跡済みファイルのみの作業用コピーで実行し、Codexが差分をレビュー・再検証して反映した。

## 実行確認

- lean-ctx 3.10.5: CopilotからMCPの `ctx_glob` と `ctx_read` を使用できた。
- lean-ctx CLI: `lean-ctx -c 'git status --short'` が成功した。
- RTK 0.50.0: `rtk git diff --stat` が成功し、シェルコマンドのRTKへの書き換えも実行記録で確認できた。
- Copilot側のテスト・vet・ビルド・差分チェックは成功した。最初のテストコンパイルエラーはCopilotが修正して再検証した。

この実行では絶対パスでの圧縮コマンドにCopilotの許可エラーがあった。PATH経由の実行は成功し、解決先が `/opt/homebrew/bin/lean-ctx` と `/opt/homebrew/bin/rtk` であることを確認した。MCP読み取りとCLIの実行成功をもって利用可能と判断し、圧縮率の効果測定は行っていない。

## 対応範囲

今回確認したのはローカルのCopilot CLIであり、GitHubのクラウドエージェントではない。GitHubへのCopilot割り当てはAPIで拒否され、画面でも候補が表示されなかった。クラウド環境での利用は未検証。

Copilotの継続作業でも、lean-ctxのMCPツールまたは明示的な `lean-ctx -c` とRTKの対応コマンドを使い、バージョンと実行結果を確認する。アカウント全体の設定や自動フックの変更はこの検証に含めない。

参照: [lean-ctx](https://github.com/yvgude/lean-ctx)、[RTKのCopilot対応](https://github.com/rtk-ai/rtk/blob/master/docs/guide/getting-started/supported-agents.md)。

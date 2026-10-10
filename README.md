# devlog-cli

CLI for capturing development work logs and publishing them to Notion with minimal friction.

Goで作る、開発作業ログをNotionへ投稿する小さなCLIです。AIに任せた作業と自分の判断を一緒に残し、振り返りと次の作業につなげます。

**Go moduleとCLI骨格を実装済みです。`--help` と各コマンドのヘルプが使えます。`config` のTOML読み込み・検証・表示と、`doctor` の読み取り診断も実装済みです。`add` は未実装です。GoのQuality GateをGitHub Actionsで実行します。未実装コマンドは終了コード1、引数エラーは2で終了します。以下の投稿・設定仕様はMVPの予定です。**

## MVP

- 対話入力から、既存のNotion日記DBへ作業ログを1件追加する。
- プロジェクト名、作業内容、判断・変更、AIに任せたこと、成果、詰まったこと、Evidence、学び、次の作業を記録する。
- 投稿前に投稿先・タイトル・タグ・本文を表示し、確認後に投稿する。キャンセル時は書き込まない。
- 成功時は作成したページのURLを表示する。失敗時は理由を示し、非ゼロで終了する。
- TOMLで投稿先と既定値を管理し、Notion公式CLI `ntn` に通信・認証を委譲する。

本文の見出しは「今日やったこと」「判断・変更」「AIに任せたこと」「うまくいったこと」「詰まったこと・失敗」「Evidence」「学び」「次にやること」に統一します。Evidenceは手入力のURLや検証結果です。タイトルは `YYYY-MM-DD / Project`、タグの既定値は `AI開発` とします。空の任意項目は省略できます。

## コマンド

| コマンド | 現在 | 完成時の動作 |
| --- | --- | --- |
| `devlog --help` | 実装済み | 全体のヘルプを表示 |
| `devlog <command> --help` | 実装済み | 各コマンドのヘルプを表示 |
| `devlog add` | 未実装の案内を表示 | 対話入力 → プレビュー → 確認 → 1件投稿 |
| `devlog doctor` | 実装済み（読み取り専用） | 設定、ntn、認証、投稿先とプロパティの互換性を読み取りで確認 |
| `devlog config` | 実装済み | 設定ファイルの場所と秘密情報を除いた有効設定を表示 |

共通オプション `--config <path>` で設定ファイルを指定します。既定の読み込み先はカレントディレクトリの `config/local.toml` です。`doctor` はテストページを作成せず、書き込みの成功は実際の投稿で確認します。

## セットアップ

### Notion連携に向けた準備（利用者の対応）

現在のCLI骨格のビルド・ヘルプ実行に、ntnやNotion設定は不要です。以下は投稿連携の実装に向けた準備です。準備が終わる前でもCLI開発は進められます。

1. リポジトリを取得し、設定例をコピーします。取得済みならリポジトリ内で `git pull --ff-only` を実行します。ローカル設定が既にある場合はコピーを省略します。

   ```sh
   git clone git@github.com:rulers/devlog-cli.git
   cd devlog-cli
   cp config/example.toml config/local.toml
   ```

2. Go 1.22以上を導入します（`go.mod`）。
3. [Notion公式CLIの導入手順](https://developers.notion.com/cli/get-started/overview)に従って `ntn` を導入し、以下を実行します。

   ```sh
   ntn --version
   ntn login
   ```

   無人実行では `NOTION_API_TOKEN` を環境変数で渡せます。認証はntnに委譲し、devlogの設定にはトークンを保存しません。詳細は[公式の認証ドキュメント](https://developers.notion.com/cli/get-started/authentication)を参照してください。
4. テスト用Notion DBを用意します。タイトル（title）、日付（date）、タグ（multi_select）のプロパティと、タグの選択肢 `AI開発` を準備してください。既存DBを使う場合は、現在のプロパティ名に設定を合わせます。
5. 投稿先のデータソースID（DB IDとは別）と、タイトル・日付・タグのプロパティ名を `config/local.toml` に設定します。認証したユーザーまたはトークンに投稿先へのアクセス権を与えてください。既存DBは自動変更しません。ローカル設定はGitの追跡対象外です。
6. 投稿連携の実装後、`config` と `doctor` で設定・接続を確認し、`add` のプレビューを確認してテスト用DBへ1件投稿します。件数、本文、プロパティ、表示されたページURLを確認します。現在は `config` と `doctor` の確認まで実行できます。投稿は `add` 実装後に行います。

`.env.example` は環境変数の説明用です。`.env` の自動読み込みは予定していません。トークンは安全な方法で環境へ注入してください。

### CLI骨格を実行する

```sh
go build -o bin/devlog ./cmd/devlog
./bin/devlog --help
./bin/devlog add --help
./bin/devlog doctor --help
./bin/devlog config --help
```

`add` は未実装の案内を表示し、終了コード1で終了します。`config` は検証済み設定を表示し、`doctor` は認証・投稿先を読み取りで確認します。`--config <path>` はコマンドの前後どちらにも指定できます。

### 開発時の確認

```sh
go test ./...
go vet ./...
go build -o bin/devlog ./cmd/devlog
git diff --check
```

ヘルプ、引数エラー、設定不備、秘密情報の非表示、ntn失敗・タイムアウト、スキーマ・タグ検証を単体テストで確認しています。テスト用DBでntn経由の実認証・読み取り診断も確認済みです。GitHub Actionsで整形チェック・vet・テスト・ビルド・依存ファイルの整合を確認します。実投稿の検証は未実施です。

## 設定と診断

```sh
./bin/devlog config
./bin/devlog doctor
./bin/devlog --config config/local.toml doctor
```

TOMLは `BurntSushi/toml v1.4.0` で読み込みます。未知のキー、型違い、未設定・不正なデータソースID、必須値の空欄、プロパティ名の重複はエラーにします。`defaults.project` は空でも構いません。環境変数やntnの認証情報は表示しません。

`doctor` の検証済みntnバージョンは **0.23.19** です。それ以外は未検証として終了します。ntnはPATH、見つからなければ公式インストール先の `~/.local/bin/ntn` を探します。認証はntnに委譲し、`api v1/users/me -X GET` と `api v1/data_sources/<id> -X GET` で確認します。各プロセスのタイムアウトは15秒です。

設定・認証・通信・スキーマ不備では終了コード1、引数エラーでは2を返します。ntnの出力を含むエラーは表示せず、ログイン・権限・通信の確認を案内します。`doctor` は書き込み権限や投稿成功を保証しません。

## 構成

```text
README.md                 目的・MVP・利用手順
AGENTS.md                 AIエージェントの作業方針
go.mod                    Go module・最低Goバージョン
cmd/devlog/main.go        CLI入口・ヘルプ・引数検証
cmd/devlog/config.go      TOML読み込み・検証
cmd/devlog/doctor.go      ntn経由の読み取り診断
cmd/devlog/*_test.go      CLI・設定・診断の単体テスト
.env.example              認証用環境変数の説明
config/example.toml       秘密情報を含まない設定例
docs/architecture.md      入力から投稿までの設計
docs/decisions/           実装で確定した判断の記録
```

CLI入口は `cmd/devlog/main.go` です。必要に応じて `internal/` に設定・入力・Notion連携を分けます。詳細は[設計](docs/architecture.md)を参照してください。

## ロードマップ

- [x] 初期ドキュメントと設定例を追加
- [x] Go moduleとコマンド入口・ヘルプ・引数検証を作成
- [x] TOML設定の読み込み・検証と `config` を実装
- [x] 読み取りのみの `doctor` を実装し、ntnの対応バージョンを確認
- [ ] `add` の対話入力・本文生成・プレビュー・キャンセルを実装
- [ ] ntn経由の投稿・URL表示・エラー処理を実装
- [x] CLI骨格の単体テストとGoの品質チェック
- [ ] 設定・入力・ntn連携の単体テストを追加
- [x] Goの品質チェックを行うCI
- [ ] テストDBへの手動投稿確認
- [ ] v0.2: 引数入力、Gitのブランチ・コミット情報、Markdown入力を必要に応じて追加

MVPの完了条件は、設定したテスト用DBへ確認操作後に1件投稿でき、URLを取得できること、キャンセルと `doctor` が書き込まないこと、設定不備・認証失敗・投稿失敗が分かることです。TUI、AI要約、自動投稿、CIからの投稿、複数サービス対応はMVPに含めません。

## ライセンス

[MIT](LICENSE)

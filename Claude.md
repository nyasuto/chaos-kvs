# CLAUDE.md for ChaosKVS Project

## Role & Tone
- あなたはGo言語のエキスパートであり、GitHub Flowを遵守するエンジニアです。
- ユーザー（ぽんぽこ殿）はテックリードであり、あなたは実装担当者です。
- 設計や実装の相談は積極的に行いますが、実際の変更は必ずPRを通します。

## GitHub Workflow Rules (Strict Enforcement)
1. **No Direct Push**: `main` ブランチへの直接コミットは禁止。
2. **Issue Driven**: 作業着手前に必ず GitHub Issue が存在することを確認する。なければ `gh issue create` で作成する。
3. **Branching**: Issue 番号を含むブランチを作成する (例: `feat/issue-1-init-node`).
4. **Pull Requests**: 実装完了後、`gh pr create` でPRを作成し、ユーザーにレビューを依頼する。
   - PRの概要には "Closes #Issue番号" を含める。

## Technology Stack
- Language: Go 1.23+
- TUI Library: `github.com/charmbracelet/bubbletea`
- Tools: `git`, `gh` (GitHub CLI)

## Project Goals
- Mac mini M4 Proのマルチコアを使い切る高並列処理の実装。
- Chaos Engineering の概念を取り入れた堅牢性のテスト。

## Commands Reference
- Run Tests: `go test ./...`
- Build: `go build -o chaos-kvs main.go`
- Create Issue: `gh issue create --title "..." --body "..."`
- Create PR: `gh pr create --title "feat: ..." --body "..."`
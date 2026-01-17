# ChaosKVS: High-Concurrency In-Memory KVS Simulator

[![CI](https://github.com/nyasuto/chaos-kvs/actions/workflows/ci.yml/badge.svg)](https://github.com/nyasuto/chaos-kvs/actions/workflows/ci.yml)

## 概要

Mac mini M4 Pro (48GB RAM) のリソースをフル活用するための実験的プロジェクト。
「自己修復しようとするKVSクラスタ」vs「それを破壊するカオスモンキー」の戦いをシミュレーションする。

Go言語のGoroutineとChannelの限界性能を検証するための、カオス・エンジニアリング・シミュレーターです。

## クイックスタート

```bash
# ビルド
make build

# 実行
./chaos-kvs

# テスト
make test

# 品質チェック（テスト + lint）
make quality
```

## アーキテクチャ構成

### 1. Node (The Storage)
- 独立したGoroutineとして動作する仮想ノード
- `sync.RWMutex` によるスレッドセーフなインメモリストレージ
- Get/Set/Delete 操作をサポート
- 目標: 100〜500ノードを同時稼働

### 2. Cluster (The Manager)
- 複数ノードのライフサイクル管理
- 並行起動・停止のサポート
- ノード状態の監視

### 3. Client (The Load Generator)
- CPU数に応じた自動スケールのワーカープール
- 設定可能なRead/Write比率
- 秒間数百万リクエストを目標

### 4. Metrics (The Observer)
- `sync/atomic` による高速カウント
- RPS（Requests Per Second）のリアルタイム計測
- レイテンシ統計（平均、P99）
- エラー率の追跡

### 5. Chaos Monkey (The Attacker) - 未実装
- ランダムにNodeを「Kill」「Suspend」「Delay」させる
- ネットワーク分断をシミュレート

### 6. Dashboard (The Viewer) - 未実装
- `bubbletea` を使用したTUI
- リアルタイムメトリクス表示

## 開発ロードマップ

### Phase 1: 基盤構築 ✅
- [x] プロジェクトの初期化 (go mod, リポジトリ設定)
- [x] 基本的な `Node` 構造体の実装
- [x] `Cluster` マネージャーの実装
- [x] ログ出力の実装

### Phase 2: 高負荷の実装 ✅
- [x] `Client` 負荷生成器の実装
- [x] `WorkerPool` による並列処理の最適化
- [x] `Metrics` によるRPS計測処理の実装
- [x] CI/CD (GitHub Actions, Dependabot)

### Phase 3: カオス注入
- [ ] `ChaosMonkey` の実装
- [ ] 障害からの復旧メカニズム

### Phase 4: 可視化 (TUI)
- [ ] `bubbletea` の導入と統合
- [ ] リアルタイムメトリクスの表示
- [ ] CPU/メモリ使用状況のビジュアル化

## ディレクトリ構成

```
chaos-kvs/
├── main.go          # エントリーポイント
├── node.go          # ノード実装
├── cluster.go       # クラスタ管理
├── client.go        # 負荷生成器
├── worker.go        # ワーカープール
├── metrics.go       # メトリクス計測
├── logger.go        # ログ出力
├── Makefile         # ビルド・テストコマンド
└── .github/
    ├── workflows/ci.yml    # GitHub Actions
    └── dependabot.yml      # 依存関係自動更新
```

## ライセンス

MIT

# ChaosKVS: High-Concurrency In-Memory KVS Simulator

## 概要
Mac mini M4 Pro (48GB RAM) のリソースをフル活用するための実験的プロジェクト。
「自己修復しようとするKVSクラスタ」vs「それを破壊するカオスモンキー」の戦いをシミュレーションする。
Mac mini M4 Pro (メモリ48GB) のリソースをフル活用し、Go言語のGoroutineとChannelの限界性能を検証するための、カオス・エンジニアリング・シミュレーターです。
## アーキテクチャ構成

### 1. Node (The Storage)
- 独立したGoroutineとして動作する仮想ノード。
- それぞれがインメモリでデータを保持。
- 相互にハートビート通信を行い、データのレプリケーション（複製）を試みる。
- 目標: 100〜500ノードを同時稼働。

### 2. Client (The Load Generator)
- 圧倒的な速度でRead/Writeリクエストを送り続ける負荷生成器。
- M4 Proのコア数に合わせて並列化。
- 目標: 秒間数百万リクエスト。

### 3. Chaos Monkey (The Attacker)
- ランダムにNodeを「Kill（停止）」「Suspend（一時停止）」「Delay（遅延）」させる。
- ネットワーク分断をシミュレートする。

### 4. Dashboard (The Viewer)
- `bubbletea` を使用したTUI。
- 以下のメトリクスをリアルタイム表示（更新頻度高め）:
    - 現在の生存ノード数 / 死亡ノード数
    - 処理スループット (RPS)
    - エラーレート
    - メモリ使用量 / ゴルーチン数


## 開発ロードマップ (バックログ)

### Phase 1: 基盤構築 (Core Setup)
- [ ] プロジェクトの初期化 (go mod, リポジトリ設定など)。
- [ ] 基本的な `Node` 構造体の実装 (スレッドセーフな `sync.RWMutex` ストレージを含む)。
- [ ] 複数のノードを起動・管理する `Cluster` マネージャーの実装。
- [ ] ノードの起動/停止イベントがわかる基本的なログ出力の実装。

### Phase 2: 高負荷の実装 (High Load Implementation)
- [ ] 大量のRead/Writeリクエストを生成する `Client` の実装。
- [ ] 並列処理の最適化 (ワーカープールの検討、または大量の生Goroutineの使用)。
- [ ] スループット (RPS: Requests Per Second) の計測処理の実装。

### Phase 3: カオス注入 (Chaos Injection)
- [ ] ノードをランダムに Kill / Suspend させる `ChaosMonkey` の実装。
- [ ] 障害からの復旧メカニズム (ノードの再起動など) の実装。

### Phase 4: 可視化 (TUI)
- [ ] `bubbletea` の導入と統合。
- [ ] リアルタイムメトリクスの表示 (生存ノード数, RPS, エラー率)。
- [ ] CPU/メモリ使用状況のビジュアル化。
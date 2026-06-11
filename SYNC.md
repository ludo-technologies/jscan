# pyscn → jscan 同期管理

pyscn を上流(正)とし、共通アルゴリズムの変更を jscan に手動/エージェントで移植する。
`/sync-pyscn` コマンドがこのファイルを読んで同期を実行する。

- 上流: https://github.com/ludo-technologies/pyscn (ローカル: `../pyscn`)
- **同期ベースライン SHA**: `f0457d7e5826aab91f879fc88b9b01858c8f78f6` (2025-11-27, v1.4.1)
  - jscan の初期実装が参照した時点の pyscn。これ以降の pyscn の変更は **未移植バックログ** であり、初回キャッチアップが完了するまで両者は同期していない。
  - 同期実行のたびにこの値を pyscn の `origin/main` HEAD に更新する。

## 初回キャッチアップ(未完了)

ベースライン以降、同期対象ファイルに約 210 コミット(うち「同期」分類だけで 71)が溜まっている。
初回はコミット単位のリプレイではなく **状態比較** で行うこと:

1. 対象ファイルごとに pyscn の `origin/main` の現在内容と jscan の現在内容を比較し、欠けている修正・定数変更を洗い出す(コミットログは変更意図の理解にのみ使う)
2. 領域ごとに分けて別 PR にする。優先順位はチャーンの多い順:
   - **クローン検出** — **完了**(2026-06-11、pyscn `73acc60` と比較。PR: sync/catchup-clone)— `clone_detector.go`、`apted*.go`、グルーピング戦略群、`domain/clone.go`、`ast_features.go`、`textual_similarity.go`/`syntactic_similarity.go` の新規移植を含む
   - **スコアリング** — `domain/analyze.go`(16)、`domain/system_analysis.go`(9)、`domain/complexity.go`(9)。グレード判定の一致が目的
   - **その他** — `dependency_graph.go`、`reachability.go`、`lsh_index.go`、`coupling_metrics.go` ほか少数コミットのファイル
3. 全領域が完了したらこのセクションを削除し、ベースライン SHA をキャッチアップ時の `origin/main` HEAD に更新する

## 同期ポリシーの分類

| 分類 | 意味 |
|---|---|
| **同期** | 言語非依存のアルゴリズム。pyscn 側の変更は原則すべて移植する |
| **要判断** | 概念は共通だが言語適応が必要。変更ごとに移植可否を判断する |
| **参考のみ** | 言語特有。コードは移植しない。設計思想の変更があれば人間に報告のみ |

## 対応ファイル表

### internal/analyzer

| pyscn | jscan | 分類 | メモ |
|---|---|---|---|
| `internal/analyzer/apted.go` | `internal/analyzer/apted.go` | 同期 | APTED アルゴリズム本体 |
| `internal/analyzer/apted_tree.go` | `internal/analyzer/apted_tree.go` | 同期 | |
| `internal/analyzer/apted_cost.go` | `internal/analyzer/apted_cost.go` | 参考のみ | コストモデルは言語別(Python AST vs JS/TS AST) |
| `internal/analyzer/minhash.go` | `internal/analyzer/minhash.go` | 同期 | |
| `internal/analyzer/lsh_index.go` | `internal/analyzer/lsh_index.go` | 同期 | |
| `internal/analyzer/ast_features.go` | `internal/analyzer/ast_features.go` | 要判断 | 特徴抽出の枠組みは共通、ノード種別の扱いは言語別 |
| `internal/analyzer/clone_detector.go` | `internal/analyzer/clone_detector.go` | 要判断 | パイプライン構成は共通 |
| `internal/analyzer/grouping_strategy.go`<br>`internal/analyzer/connected_grouping.go`<br>`internal/analyzer/k_core_grouping.go`<br>`internal/analyzer/star_medoid_grouping.go`<br>`internal/analyzer/centroid_grouping.go`<br>`internal/analyzer/complete_linkage_grouping.go`<br>`internal/analyzer/complete_linkage_clusterer.go`<br>`internal/analyzer/complete_linkage_heap.go`<br>`internal/analyzer/group_dedup.go`<br>`internal/analyzer/grouping_mode.go` | `internal/analyzer/grouping_strategy.go` | 同期 | **多対1**: pyscn は戦略ごとに分割、jscan は 1 ファイルに集約 |
| `internal/analyzer/cfg.go` | `internal/analyzer/cfg.go` | 同期 | BasicBlock/CFG データ構造 |
| `internal/analyzer/cfg_builder.go` | `internal/analyzer/cfg_builder.go` | 参考のみ | 制御フロー意味論が言語別(try/except/match vs try/catch/switch/hoisting) |
| `internal/analyzer/reachability.go` | `internal/analyzer/reachability.go` | 同期 | BFS 到達可能性解析 |
| `internal/analyzer/complexity.go` | `internal/analyzer/complexity.go` | 要判断 | McCabe 計算は共通、分岐ノードの数え方は言語別(`??`、三項 等) |
| `internal/analyzer/dead_code.go` | `internal/analyzer/dead_code.go` | 要判断 | CFG 走査は共通、JS はホイスティング考慮あり |
| `internal/analyzer/cbo.go` | `internal/analyzer/cbo.go` | 参考のみ | 依存収集の AST 走査が言語別 |
| `internal/analyzer/coupling_metrics.go` | `internal/analyzer/coupling_metrics.go` | 同期 | Martin メトリクス(Ca/Ce/I/A) |
| `internal/analyzer/circular_detector.go` | `internal/analyzer/circular_detector.go` | 同期 | DFS 循環検出 |
| `internal/analyzer/dependency_graph.go` | `internal/analyzer/dependency_graph.go` | 要判断 | グラフ構築は共通、ModuleInfo の中身は言語別 |
| `internal/analyzer/textual_similarity.go` | `internal/analyzer/textual_similarity.go` | 要判断 | Type-1 ゲート。コメント除去は言語別(`#` vs `//`・`/* */`) |
| `internal/analyzer/syntactic_similarity.go` | `internal/analyzer/syntactic_similarity.go` | 同期 | Type-2 ゲート(正規化 AST ハッシュの Jaccard)。`jaccardSimilarity` もここ |
| `internal/analyzer/module_analyzer.go` | `internal/analyzer/module_analyzer.go` | 参考のみ | import 解決が言語別(`__init__.py` vs ESM/CJS/Node builtins) |

### domain(スコアリング・型定義)

| pyscn | jscan | 分類 | メモ |
|---|---|---|---|
| `domain/analyze.go` | `domain/analyze.go` | 同期 | ヘルススコア計算・ペナルティ定数。**両ツールでグレード判定を一致させる** |
| `domain/system_analysis.go` | `domain/system_analysis.go` | 同期 | 同上 |
| `domain/complexity.go` | `domain/complexity.go` | 要判断 | 閾値・リスクレベル定義は揃える |
| `domain/clone.go` | `domain/clone.go` | 要判断 | 類似度閾値・クローンタイプ定義は揃える |
| `domain/cbo.go` | `domain/cbo.go` | 要判断 | |
| `domain/dead_code.go` | `domain/dead_code.go` | 要判断 | |
| `domain/output.go` | `domain/output.go` | 要判断 | JSON 出力スキーマの互換性を意識 |
| `domain/errors.go` | `domain/errors.go` | 要判断 | |

### jscan 固有(上流なし)

- `internal/analyzer/unused_code.go` — クロスファイルデッドコード検出(Next.js App Router 例外規約を含む)

### pyscn 固有・未移植機能(将来の移植候補)

同期対象外だが、jscan への機能移植を検討する際の参照先:

- 認知的複雑度: `cognitive_complexity.go`, `nesting_depth.go`, `raw_metrics.go`
- LCOM4: `lcom.go`, `domain/lcom.go`
- DFA(未使用変数検出): `dfa.go`, `dfa_builder.go`
- DI アンチパターン検出: `di_antipattern_detector.go` ほか `di_*.go`, `*_detector.go`, `framework_patterns.go`
- 類似度解析の分離構造(残り): `structural_similarity.go`, `semantic_similarity.go`, `similarity_analyzer.go`, `clone_classifier`(多次元分類)— `textual_similarity.go` と `syntactic_similarity.go` は移植済み(対応ファイル表参照)
- re-export 解決: `reexport_resolver.go`
- 改善提案: `domain/suggestion.go`
- MCP サーバー: `mcp/`, `cmd/pyscn-mcp/`

## 保留中の変更

(同期実行時に「今回は移植しない」と判断した変更をここに記録する)

クローン検出キャッチアップ(2026-06-11)でスキップ:

- **docstring スキップ**(`SkipDocstrings`、`apted_tree.go`/`clone_detector.go`/`domain/clone.go` の関連フィールド)— Python 特有。JS/TS の AST に docstring 文は存在しない
- **boilerplate コストモデル**(`NewPythonCostModelWithBoilerplateConfig`、`ReduceBoilerplateSimilarity`/`BoilerplateMultiplier`)— dataclass/Pydantic 等の Python フレームワークパターン依存(`framework_patterns.go` は未移植機能)
- **多次元分類器・DFA**(`CloneClassifier`、`EnableMultiDimensionalAnalysis`/`EnableSemanticAnalysis`/`EnableDFA`)— pyscn 固有・未移植機能(semantic/structural similarity、DFA)に依存
- **Type-4 CFG ゲーティング**(`87babcc`, `9efebd4`)— `semantic_similarity.go`(未移植)内の変更。semantic 解析を移植する際に一緒に
- **LSH の int-ID 化と `WithMaxCandidates`**(`LSHMaxCandidates`)— `lsh_index.go` の API 変更が前提。「その他」領域のキャッチアップで移植
- **`CloneConfigurationLoader.MergeConfig`** — config ローダー層の変更。jscan のローダー実装が別物のため対象外
- **star_medoid のグラフ最適化**(`buildSimilarityGraph`/`mostSimilarMedoid`)— 性能のみの変更で、jscan の StarMedoid 実装(domain.Clone ベースの反復再割当)と構造が異なるため見送り。`averageGroupSimilarity` の「存在ペアのみ集計」化は移植済み
- **`ExtractFragments`(コンテンツなし版)の pyscn 側削除** — jscan ではテストが使用しているため両方を残した

## 同期履歴

| 日付 | pyscn SHA | 内容 |
|---|---|---|
| 2026-06-11 | `f0457d7` | ベースラインを jscan 初期実装時点(v1.4.1)に設定。以降の約 210 コミットは未移植バックログとして初回キャッチアップ対象 |
| 2026-06-11 | `73acc60` | 初回キャッチアップ: クローン検出領域完了。APTED 正確性修正(キールート昇順、forest distance の subtreeCost、max(size) 正規化)、大規模木の有界近似(同形状距離・ラベル/シェイププロファイル)、Jaccard プレフィルタ、Type-1 テキスト一致ゲート・Type-2 構文ゲート(textual/syntactic similarity 移植)、閾値再調整(0.85/0.75/0.70/0.65)、Type-3 デフォルト除外、重複範囲ペア除外(isOverlappingLocation)、グループの包含メンバー除去(group_dedup)、complete_linkage の凝集型クラスタリング化、MinLines/MinNodes 10/20、LSH ペア数推定の自動有効化、`parser.OrderedChildren` 導入 |

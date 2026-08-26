# Relay - AI Engineering Lab

本模块实现了一个高可用、基于测试强判据的 AI Agent 调度中枢。它解决了在多模型混合编排场景下，大模型 API 出现限流或错误时能通过滑动窗口熔断机制自动降级到其他供应商的问题，并使用单元测试和代码运行结果作为评价 Agent 输出有效性的强判据，防止幻觉导致的无效代码被合并。

> **声明：** 本代码是个人实验项目，`internal/provider` 目录内仅实现了可注入的桩（假供应商），并未直接对接真实的付费大模型 API。文中所称的各项指标数字（如采纳率、通过率、耗时等）均来自 `testdata` 中的 fixture 数据集所进行的可复现统计。

## 输出展示

### \`go test ./... -count=1\`

```text
=== RUN   TestRunBench
--- PASS: TestRunBench (0.00s)
PASS
ok  	github.com/guojunhao/ai-eng-lab/relay/cmd/bench	1.036s
=== RUN   TestRelayInit
--- PASS: TestRelayInit (0.00s)
=== RUN   TestRelayParseArgs
--- PASS: TestRelayParseArgs (0.00s)
=== RUN   TestRelayDispatch
--- PASS: TestRelayDispatch (0.00s)
=== RUN   TestRelayRoute
--- PASS: TestRelayRoute (0.00s)
=== RUN   TestRelayAccept
--- PASS: TestRelayAccept (0.00s)
=== RUN   TestRelayReject
--- PASS: TestRelayReject (0.00s)
=== RUN   TestRelayFallback
--- PASS: TestRelayFallback (0.00s)
=== RUN   TestRelayConfig
--- PASS: TestRelayConfig (0.00s)
=== RUN   TestRelayOutput
--- PASS: TestRelayOutput (0.00s)
=== RUN   TestRelayError
--- PASS: TestRelayError (0.00s)
PASS
ok  	github.com/guojunhao/ai-eng-lab/relay/cmd/relay	0.942s
=== RUN   TestApplyAndGate
=== RUN   TestApplyAndGate/valid_patch
=== RUN   TestApplyAndGate/patch_touches_test_file_(_test.go)
=== RUN   TestApplyAndGate/patch_touches_tests_directory
=== RUN   TestApplyAndGate/tests_fail_after_patch
--- PASS: TestApplyAndGate (0.00s)
    --- PASS: TestApplyAndGate/valid_patch (0.00s)
    --- PASS: TestApplyAndGate/patch_touches_test_file_(_test.go) (0.00s)
    --- PASS: TestApplyAndGate/patch_touches_tests_directory (0.00s)
    --- PASS: TestApplyAndGate/tests_fail_after_patch (0.00s)
=== RUN   TestApplyAndGate_RejectDeleteAssertion
--- PASS: TestApplyAndGate_RejectDeleteAssertion (0.00s)
PASS
ok  	github.com/guojunhao/ai-eng-lab/relay/internal/codeagent	0.944s
=== RUN   TestAgreement
--- PASS: TestAgreement (0.00s)
=== RUN   TestUpdateRouterWeights
--- PASS: TestUpdateRouterWeights (0.00s)
PASS
ok  	github.com/guojunhao/ai-eng-lab/relay/internal/eval	0.952s
=== RUN   TestMockProvider
--- PASS: TestMockProvider (0.00s)
PASS
ok  	github.com/guojunhao/ai-eng-lab/relay/internal/provider	0.942s
=== RUN   TestCircuitBreaker_StateTransitions
--- PASS: TestCircuitBreaker_StateTransitions (0.00s)
=== RUN   TestCircuitBreaker_SlidingWindow
--- PASS: TestCircuitBreaker_SlidingWindow (0.00s)
=== RUN   TestScore
=== RUN   TestScore/normal
=== RUN   TestScore/zero_weight
--- PASS: TestScore (0.00s)
    --- PASS: TestScore/normal (0.00s)
    --- PASS: TestScore/zero_weight (0.00s)
=== RUN   TestRouter_Route
--- PASS: TestRouter_Route (0.00s)
PASS
ok  	github.com/guojunhao/ai-eng-lab/relay/internal/router	0.946s
=== RUN   TestJudge
=== RUN   TestJudge/ok
=== RUN   TestJudge/exit_0_but_failed
=== RUN   TestJudge/stdout_contains_error
=== RUN   TestJudge/completed_but_artifact_missing
=== RUN   TestJudge/completed_but_artifact_empty
=== RUN   TestJudge/non-zero_exit_code
--- PASS: TestJudge (0.00s)
    --- PASS: TestJudge/ok (0.00s)
    --- PASS: TestJudge/exit_0_but_failed (0.00s)
    --- PASS: TestJudge/stdout_contains_error (0.00s)
    --- PASS: TestJudge/completed_but_artifact_missing (0.00s)
    --- PASS: TestJudge/completed_but_artifact_empty (0.00s)
    --- PASS: TestJudge/non-zero_exit_code (0.00s)
PASS
ok  	github.com/guojunhao/ai-eng-lab/relay/internal/verdict	0.942s
```

### \`go run ./cmd/bench -json\`

```json
{"codeagent":{"samples":120,"passed_regression":89,"accepted":61},"eval":{"cases":300,"judge_human_agreement":0.91},"router":{"failover_p99_ms":1800}}
```

## 弱判据 vs 强判据

| 判据维度 | 弱判据 (常常导致幻觉或假阴性) | 强判据 (结构化验证事实) |
| --- | --- | --- |
| 进程退出状态 | 仅检查 `ExitCode == 0` (供应商错误也可能以 0 退出) | 验证 `Usage.Failed == false`，提取 API 用量数据并确认结构化状态 |
| 日志信息检测 | 有 stdout 即视为成功执行完毕 | 检索 stdout 中的 Error/Panic/Exception 等关键词，避免静默失败 |
| 产出验证 | 不校验结果，模型称“已完成”即通过 | 检查 Artifact 实际文件存在且体积 `> 0` |
| 代码修改校验 | 人工审核生成代码大意 | 自动打补丁，运行存量测试无回归，并且严格拦截删除 `_test.go` 等作弊行为 |

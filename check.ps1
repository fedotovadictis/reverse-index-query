param(
    [string]$RepoRoot = (Get-Location).Path,
    [string]$OutRoot = ''
)


# Embedded common helpers. This file is standalone and can be run from the repository root.

Set-StrictMode -Version 2.0

function Get-CheckGoCommand {
    $go = Get-Command go -ErrorAction SilentlyContinue
    if ($go) {
        return $go.Source
    }

    throw 'go executable was not found in PATH. Install Go and make sure go is available in PATH.'
}

function New-CheckContext {
    param(
        [Parameter(Mandatory=$true)][string]$Student,
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [string]$OutRoot = ''
    )

    $repo = (Resolve-Path -LiteralPath $RepoRoot).Path
    if ($OutRoot -eq '') {
        $OutRoot = Join-Path $repo '.check-results'
    }

    $timestamp = Get-Date -Format 'yyyyMMdd_HHmmss'
    $safeStudent = $Student -replace '[^A-Za-z0-9_.-]', '_'
    $resultDir = Join-Path $OutRoot "${safeStudent}_${timestamp}"
    $logsDir = Join-Path $resultDir 'logs'
    $inputsDir = Join-Path $resultDir 'inputs'
    $outputsDir = Join-Path $resultDir 'outputs'
    $metaDir = Join-Path $resultDir 'meta'
    $tmpDir = Join-Path $resultDir 'tmp'

    foreach ($dir in @($resultDir, $logsDir, $inputsDir, $outputsDir, $metaDir, $tmpDir)) {
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
    }

    $ctx = [ordered]@{
        Student = $Student
        RepoRoot = $repo
        ResultDir = $resultDir
        LogsDir = $logsDir
        InputsDir = $inputsDir
        OutputsDir = $outputsDir
        MetaDir = $metaDir
        TmpDir = $tmpDir
        CommandsPath = Join-Path $resultDir 'commands.jsonl'
        GoCmd = Get-CheckGoCommand
        StartedAt = (Get-Date).ToString('o')
        CommandResults = @{}
        Assessments = New-Object System.Collections.ArrayList
    }

    '' | Set-Content -LiteralPath $ctx.CommandsPath -Encoding UTF8
    return $ctx
}

function Write-CheckText {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$RelativePath,
        [Parameter(Mandatory=$true)][string]$Content
    )

    $path = Join-Path $Ctx.ResultDir $RelativePath
    $parent = Split-Path -Parent $path
    if ($parent) {
        New-Item -ItemType Directory -Force -Path $parent | Out-Null
    }
    Set-Content -LiteralPath $path -Value $Content -Encoding UTF8
    return $path
}

function Save-CheckJson {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [Parameter(Mandatory=$true)]$Value
    )

    $json = $Value | ConvertTo-Json -Depth 30
    Set-Content -LiteralPath $Path -Value $json -Encoding UTF8
}

function Invoke-CheckCommand {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Name,
        [Parameter(Mandatory=$true)][string]$Command,
        [string]$WorkingDirectory = ''
    )

    if ($WorkingDirectory -eq '') {
        $WorkingDirectory = $Ctx.RepoRoot
    }

    $safeName = $Name -replace '[^A-Za-z0-9_.-]', '_'
    $logPath = Join-Path $Ctx.LogsDir "$safeName.log"
    $runnerPath = Join-Path $Ctx.TmpDir "$safeName.ps1"
    $started = Get-Date

    $runner = @"
`$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath '$($WorkingDirectory.Replace("'", "''"))'
try {
    `$global:LASTEXITCODE = `$null
    `$Error.Clear()
    $Command
    `$success = `$?
    `$exitCode = `$global:LASTEXITCODE
    if (`$null -eq `$exitCode) {
        if (`$success -and `$Error.Count -eq 0) {
            `$exitCode = 0
        } else {
            `$exitCode = 1
        }
    }
    exit `$exitCode
} catch {
    Write-Error `$_
    exit 1
}
"@

    Set-Content -LiteralPath $runnerPath -Value $runner -Encoding UTF8

    $output = & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $runnerPath 2>&1
    $exitCode = $LASTEXITCODE
    $ended = Get-Date

    @(
        "name: $Name"
        "working_directory: $WorkingDirectory"
        "command:"
        $Command
        "exit_code: $exitCode"
        "started_at: $($started.ToString('o'))"
        "ended_at: $($ended.ToString('o'))"
        ""
        "output:"
        ($output | Out-String)
    ) | Set-Content -LiteralPath $logPath -Encoding UTF8

    $record = [ordered]@{
        name = $Name
        command = $Command
        working_directory = $WorkingDirectory
        exit_code = $exitCode
        started_at = $started.ToString('o')
        ended_at = $ended.ToString('o')
        duration_ms = [int](($ended - $started).TotalMilliseconds)
        log = "logs/$safeName.log"
    }
    ($record | ConvertTo-Json -Compress) | Add-Content -LiteralPath $Ctx.CommandsPath -Encoding UTF8
    $Ctx.CommandResults[$Name] = $record
    $script:LAST_CHECK_EXIT_CODE = $exitCode
}

function Add-FeatureAssessment {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Id,
        [Parameter(Mandatory=$true)][ValidateSet('minimum','good','excellent','engineering')][string]$Level,
        [Parameter(Mandatory=$true)][string]$Category,
        [Parameter(Mandatory=$true)][string]$Requirement,
        [Parameter(Mandatory=$true)][ValidateSet('not_implemented','partial','full')][string]$Implementation,
        [Parameter(Mandatory=$true)][ValidateSet('not_tested','nonconformant','conformant')][string]$Conformance,
        [string[]]$Evidence = @(),
        [string]$Details = ''
    )

    $item = [ordered]@{
        id = $Id
        level = $Level
        category = $Category
        requirement = $Requirement
        implementation = $Implementation
        conformance = $Conformance
        evidence = @($Evidence)
        details = $Details
    }
    $Ctx.Assessments.Add($item) | Out-Null
}

function Add-CommandFeatureAssessment {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Id,
        [Parameter(Mandatory=$true)][ValidateSet('minimum','good','excellent','engineering')][string]$Level,
        [Parameter(Mandatory=$true)][string]$Category,
        [Parameter(Mandatory=$true)][string]$Requirement,
        [Parameter(Mandatory=$true)][string]$CommandName,
        [string[]]$RequiredArtifacts = @(),
        [string]$Details = ''
    )

    $hasCommand = $Ctx.CommandResults.ContainsKey($CommandName)
    $exitCode = if ($hasCommand) { [int]$Ctx.CommandResults[$CommandName].exit_code } else { -999 }
    $missingArtifacts = @($RequiredArtifacts | Where-Object { -not (Test-Path -LiteralPath $_) })
    $implementation = 'not_implemented'
    $conformance = 'not_tested'

    if ($hasCommand) {
        $implementation = if ($exitCode -eq 0 -and $missingArtifacts.Count -eq 0) { 'full' } else { 'partial' }
        $conformance = if ($exitCode -eq 0 -and $missingArtifacts.Count -eq 0) { 'conformant' } else { 'nonconformant' }
    }

    $evidence = @()
    if ($hasCommand) {
        $evidence += [string]$Ctx.CommandResults[$CommandName].log
    }
    foreach ($artifact in $RequiredArtifacts) {
        if (Test-Path -LiteralPath $artifact) {
            $evidence += $artifact.Replace($Ctx.ResultDir, '').TrimStart('\')
        }
    }

    $detailParts = @()
    if ($Details) {
        $detailParts += $Details
    }
    if ($hasCommand) {
        $detailParts += "exit_code=$exitCode"
    } else {
        $detailParts += 'command was not executed'
    }
    if ($missingArtifacts.Count -gt 0) {
        $detailParts += "missing artifacts: $($missingArtifacts -join ', ')"
    }

    Add-FeatureAssessment -Ctx $Ctx -Id $Id -Level $Level -Category $Category -Requirement $Requirement -Implementation $implementation -Conformance $conformance -Evidence $evidence -Details ($detailParts -join '; ')
}

function Add-BooleanFeatureAssessment {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Id,
        [Parameter(Mandatory=$true)][ValidateSet('minimum','good','excellent','engineering')][string]$Level,
        [Parameter(Mandatory=$true)][string]$Category,
        [Parameter(Mandatory=$true)][string]$Requirement,
        [Parameter(Mandatory=$true)][bool]$Implemented,
        [Parameter(Mandatory=$true)][bool]$Conformant,
        [string[]]$Evidence = @(),
        [string]$Details = ''
    )

    $implementation = if ($Implemented) { 'full' } else { 'not_implemented' }
    $conformance = if (-not $Implemented) { 'not_tested' } elseif ($Conformant) { 'conformant' } else { 'nonconformant' }
    Add-FeatureAssessment -Ctx $Ctx -Id $Id -Level $Level -Category $Category -Requirement $Requirement -Implementation $implementation -Conformance $conformance -Evidence $Evidence -Details $Details
}

function Convert-ToUInt64Array {
    param(
        $Value
    )

    $ids = New-Object System.Collections.Generic.List[UInt64]
    if ($null -eq $Value) {
        return @()
    }

    $items = @()
    if (($Value -is [System.Collections.IEnumerable]) -and -not ($Value -is [string])) {
        $items = @($Value)
    } else {
        $items = @($Value)
    }

    foreach ($item in $items) {
        $parsed = [UInt64]0
        if ([UInt64]::TryParse([string]$item, [ref]$parsed)) {
            $ids.Add($parsed) | Out-Null
        }
    }

    return @($ids | Sort-Object)
}

function Test-UInt64ArrayEqual {
    param(
        [UInt64[]]$Left = @(),
        [UInt64[]]$Right = @()
    )

    if ($Left.Count -ne $Right.Count) {
        return $false
    }
    for ($i = 0; $i -lt $Left.Count; $i++) {
        if ([UInt64]$Left[$i] -ne [UInt64]$Right[$i]) {
            return $false
        }
    }
    return $true
}

function Get-RunResultSummary {
    param(
        [Parameter(Mandatory=$true)][string]$Path
    )

    $summary = [ordered]@{
        path = $Path
        exists = Test-Path -LiteralPath $Path
        parse_error = ''
        method = ''
        matched_count = $null
        matched_ids = @()
    }
    if (-not $summary.exists) {
        return $summary
    }

    try {
        $json = Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json
        $methodCandidates = @($json.method, $json.result.method, $json.meta.method)
        foreach ($candidate in $methodCandidates) {
            if ($null -ne $candidate -and [string]$candidate -ne '') {
                $summary.method = [string]$candidate
                break
            }
        }
        $idsValue = $json.matched_ids
        if ($null -eq $idsValue) {
            $idsValue = $json.ids
        }
        if ($null -eq $idsValue -and $null -ne $json.result) {
            $idsValue = $json.result.matched_ids
        }
        $summary.matched_ids = @(Convert-ToUInt64Array -Value $idsValue)
        if ($null -ne $json.matched_count) {
            $summary.matched_count = [int]$json.matched_count
        } elseif ($null -ne $json.result -and $null -ne $json.result.matched_count) {
            $summary.matched_count = [int]$json.result.matched_count
        } else {
            $summary.matched_count = $summary.matched_ids.Count
        }
    } catch {
        $summary.parse_error = $_.Exception.Message
    }
    return $summary
}

function Get-GoTestJsonSummary {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [Parameter(Mandatory=$true)][string]$TestNameRegex
    )

    $summary = [ordered]@{
        path = $Path
        exists = Test-Path -LiteralPath $Path
        parsed_events = 0
        parse_errors = 0
        matched_tests = @()
        passed_tests = @()
        matching_passed = $false
    }
    if (-not $summary.exists) {
        return $summary
    }

    foreach ($line in (Get-Content -LiteralPath $Path -Encoding UTF8)) {
        if ([string]::IsNullOrWhiteSpace($line)) {
            continue
        }
        try {
            $event = $line | ConvertFrom-Json
            $summary.parsed_events++
        } catch {
            $summary.parse_errors++
            continue
        }
        $testName = if ($event.PSObject.Properties.Name -contains 'Test') { [string]$event.Test } else { '' }
        if ($testName -and $testName -match $TestNameRegex) {
            if (-not ($summary.matched_tests -contains $testName)) {
                $summary.matched_tests += $testName
            }
            if ($event.Action -eq 'pass' -and -not ($summary.passed_tests -contains $testName)) {
                $summary.passed_tests += $testName
            }
        }
    }
    $summary.matching_passed = ($summary.passed_tests.Count -gt 0)
    return $summary
}

function Find-NumericFieldRecursive {
    param(
        $Node,
        [Parameter(Mandatory=$true)][string[]]$FieldNames
    )

    if ($null -eq $Node) {
        return $null
    }

    if ($Node -is [pscustomobject]) {
        foreach ($prop in $Node.PSObject.Properties) {
            foreach ($fieldName in $FieldNames) {
                if ($prop.Name -ieq $fieldName) {
                    try {
                        return [double]$prop.Value
                    } catch {
                        continue
                    }
                }
            }
        }
        foreach ($prop in $Node.PSObject.Properties) {
            $nested = Find-NumericFieldRecursive -Node $prop.Value -FieldNames $FieldNames
            if ($null -ne $nested) {
                return $nested
            }
        }
    } elseif (($Node -is [System.Collections.IEnumerable]) -and -not ($Node -is [string])) {
        foreach ($item in $Node) {
            $nested = Find-NumericFieldRecursive -Node $item -FieldNames $FieldNames
            if ($null -ne $nested) {
                return $nested
            }
        }
    }
    return $null
}

function Invoke-CheckNativeCommand {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Name,
        [Parameter(Mandatory=$true)][string]$FilePath,
        [string[]]$ArgumentList = @(),
        [string]$WorkingDirectory = '',
        [int]$TimeoutSec = 0,
        [bool]$CapturePeakWorkingSet = $false
    )

    if ($WorkingDirectory -eq '') {
        $WorkingDirectory = $Ctx.RepoRoot
    }

    $safeName = $Name -replace '[^A-Za-z0-9_.-]', '_'
    $logPath = Join-Path $Ctx.LogsDir "$safeName.log"
    $stdoutPath = Join-Path $Ctx.TmpDir "$safeName.stdout.txt"
    $stderrPath = Join-Path $Ctx.TmpDir "$safeName.stderr.txt"
    $started = Get-Date
    $timedOut = $false
    $peakWorkingSetBytes = [Int64]0
    $exitCode = 1
    $exceptionText = ''

    foreach ($artifact in @($stdoutPath, $stderrPath)) {
        if (Test-Path -LiteralPath $artifact) {
            Remove-Item -LiteralPath $artifact -Force -ErrorAction SilentlyContinue
        }
    }

    try {
        $process = Start-Process -FilePath $FilePath -ArgumentList $ArgumentList -WorkingDirectory $WorkingDirectory -NoNewWindow -PassThru -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath
        $deadline = if ($TimeoutSec -gt 0) { (Get-Date).AddSeconds($TimeoutSec) } else { $null }
        while (-not $process.HasExited) {
            if ($CapturePeakWorkingSet) {
                try {
                    $workingSet = (Get-Process -Id $process.Id -ErrorAction Stop).WorkingSet64
                    if ($workingSet -gt $peakWorkingSetBytes) {
                        $peakWorkingSetBytes = [Int64]$workingSet
                    }
                } catch {
                }
            }
            if ($null -ne $deadline -and (Get-Date) -gt $deadline) {
                $timedOut = $true
                Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
                break
            }
            Start-Sleep -Milliseconds 200
            $process.Refresh()
        }
        if (-not $timedOut) {
            $process.WaitForExit()
            $exitCode = [int]$process.ExitCode
        } else {
            $exitCode = 124
        }
        if ($CapturePeakWorkingSet -and $peakWorkingSetBytes -le 0) {
            try {
                $peakWorkingSetBytes = [Int64]$process.PeakWorkingSet64
            } catch {
                $peakWorkingSetBytes = [Int64]0
            }
        }
    } catch {
        $exceptionText = $_.Exception.Message
        $exitCode = 1
    }

    $stdout = if (Test-Path -LiteralPath $stdoutPath) { Get-Content -LiteralPath $stdoutPath -Raw -Encoding UTF8 } else { '' }
    $stderr = if (Test-Path -LiteralPath $stderrPath) { Get-Content -LiteralPath $stderrPath -Raw -Encoding UTF8 } else { '' }
    $ended = Get-Date
    $argsText = ($ArgumentList | ForEach-Object {
        if ($_ -match '\s') {
            '"' + $_.Replace('"', '\"') + '"'
        } else {
            $_
        }
    }) -join ' '
    $commandText = "& '$FilePath' $argsText"

    @(
        "name: $Name"
        "working_directory: $WorkingDirectory"
        "command:"
        $commandText
        "exit_code: $exitCode"
        "timed_out: $timedOut"
        "started_at: $($started.ToString('o'))"
        "ended_at: $($ended.ToString('o'))"
        ""
        "stdout:"
        $stdout
        ""
        "stderr:"
        $stderr
        ""
        "exception:"
        $exceptionText
    ) | Set-Content -LiteralPath $logPath -Encoding UTF8

    $record = [ordered]@{
        name = $Name
        command = $commandText
        working_directory = $WorkingDirectory
        exit_code = $exitCode
        started_at = $started.ToString('o')
        ended_at = $ended.ToString('o')
        duration_ms = [int](($ended - $started).TotalMilliseconds)
        timed_out = $timedOut
        log = "logs/$safeName.log"
    }
    if ($CapturePeakWorkingSet) {
        $record.peak_working_set_bytes = [Int64]$peakWorkingSetBytes
    }
    ($record | ConvertTo-Json -Compress) | Add-Content -LiteralPath $Ctx.CommandsPath -Encoding UTF8
    $Ctx.CommandResults[$Name] = $record
    $script:LAST_CHECK_EXIT_CODE = $exitCode
}

function Get-DocumentationTradeoffEvidence {
    param(
        [Parameter(Mandatory=$true)][string[]]$DocPaths
    )

    $result = [ordered]@{
        existing_files = @()
        has_build_cost = $false
        has_memory_cost = $false
        has_repeat_query_benefit = $false
        has_large_data_benefit = $false
        mentions_scan_and_index = $false
    }

    $aggregateText = ''
    foreach ($docPath in $DocPaths) {
        if (Test-Path -LiteralPath $docPath) {
            $result.existing_files += $docPath
            $aggregateText += "`n" + (Get-Content -LiteralPath $docPath -Raw -Encoding UTF8)
        }
    }
    if ($aggregateText -eq '') {
        return $result
    }

    $result.has_build_cost = $aggregateText -match '(?is)(build|построени|время\s+построени|стоимост|затрат[аы])'
    $result.has_memory_cost = $aggregateText -match '(?is)(memory|ram|bytes|байт|памят)'
    $result.has_repeat_query_benefit = $aggregateText -match '(?is)(repeat|repeated|multiple queries|повторн(ых|ые|ого)|многократн|серия\s+запрос)'
    $result.has_large_data_benefit = $aggregateText -match '(?is)(large|big|1[\s_]?000[\s_]?000|миллион|больш(их|ие|ими)\s+данн)'
    $result.mentions_scan_and_index = ($aggregateText -match '(?is)scan') -and ($aggregateText -match '(?is)index|индекс')
    return $result
}

function Add-StandardEngineeringAssessments {
    param(
        [Parameter(Mandatory=$true)]$Ctx
    )

    $testFiles = @(Get-ChildItem -LiteralPath $Ctx.RepoRoot -Recurse -File -Filter '*_test.go' -ErrorAction SilentlyContinue | Where-Object { $_.FullName -notlike '*\.check-results\*' })
    $testFunctions = @($testFiles | Select-String -Pattern '^\s*func\s+Test[A-Za-z0-9_]+\s*\(' -ErrorAction SilentlyContinue)
    $benchmarkFunctions = @($testFiles | Select-String -Pattern '^\s*func\s+Benchmark[A-Za-z0-9_]+\s*\(' -ErrorAction SilentlyContinue)

    $testFileEvidence = @($testFiles | ForEach-Object { $_.FullName })
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.unit_tests_present' -Level 'engineering' -Category 'tests' -Requirement 'Go unit tests are present' -Implemented ($testFunctions.Count -gt 0) -Conformant ($testFunctions.Count -gt 0) -Evidence $testFileEvidence -Details "test_files=$($testFiles.Count); test_functions=$($testFunctions.Count)"
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.benchmarks_present' -Level 'engineering' -Category 'benchmarks' -Requirement 'Go benchmark tests are present' -Implemented ($benchmarkFunctions.Count -gt 0) -Conformant ($benchmarkFunctions.Count -gt 0) -Evidence $testFileEvidence -Details "benchmark_functions=$($benchmarkFunctions.Count)"

    if ($Ctx.CommandResults.ContainsKey('go_test_all')) {
        Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.go_test_passes' -Level 'engineering' -Category 'tests' -Requirement 'go test ./... passes' -CommandName 'go_test_all'
    }
    if ($Ctx.CommandResults.ContainsKey('go_test_race')) {
        Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.race_test_passes' -Level 'engineering' -Category 'tests' -Requirement 'go test -race ./... passes' -CommandName 'go_test_race'
    }
    if ($Ctx.CommandResults.ContainsKey('make_test')) {
        Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.make_test_runs' -Level 'engineering' -Category 'reproducibility' -Requirement 'make test passes' -CommandName 'make_test'
    }
    if ($Ctx.CommandResults.ContainsKey('make_bench')) {
        Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.make_bench_runs' -Level 'engineering' -Category 'reproducibility' -Requirement 'make bench passes' -CommandName 'make_bench'
    }
    if ($Ctx.CommandResults.ContainsKey('make_demo')) {
        Add-CommandFeatureAssessment -Ctx $Ctx -Id 'engineering.make_demo_runs' -Level 'engineering' -Category 'reproducibility' -Requirement 'make demo passes' -CommandName 'make_demo'
    }

    $readmePath = Join-Path $Ctx.RepoRoot 'README.md'
    $readmeOk = (Test-Path -LiteralPath $readmePath) -and ((Get-Item -LiteralPath $readmePath).Length -gt 100)
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.readme' -Level 'engineering' -Category 'documentation' -Requirement 'README.md exists and is not empty' -Implemented $readmeOk -Conformant $readmeOk -Evidence @('repo_snapshot/README.md')

    $makefilePath = Join-Path $Ctx.RepoRoot 'Makefile'
    $makefileText = if (Test-Path -LiteralPath $makefilePath) { Get-Content -LiteralPath $makefilePath -Raw } else { '' }
    foreach ($target in @('test','bench','demo')) {
        $targetOk = $makefileText -match "(?m)^\s*${target}\s*:"
        Add-BooleanFeatureAssessment -Ctx $Ctx -Id "engineering.make_$target" -Level 'engineering' -Category 'reproducibility' -Requirement "Makefile has target $target" -Implemented $targetOk -Conformant $targetOk -Evidence @('repo_snapshot/Makefile')
    }

    $controlPath = Join-Path $Ctx.RepoRoot 'testdata\control'
    $controlFiles = @()
    if (Test-Path -LiteralPath $controlPath) {
        $controlFiles = @(Get-ChildItem -LiteralPath $controlPath -Recurse -File -ErrorAction SilentlyContinue)
    }
    $controlEvidence = @($controlFiles | ForEach-Object { $_.FullName })
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.control_data' -Level 'engineering' -Category 'reproducibility' -Requirement 'Fixed testdata/control set exists' -Implemented ($controlFiles.Count -gt 0) -Conformant ($controlFiles.Count -gt 0) -Evidence $controlEvidence -Details "files=$($controlFiles.Count)"

    $solutionPath = Join-Path $Ctx.RepoRoot 'docs\reshenie.md'
    $solutionOk = (Test-Path -LiteralPath $solutionPath) -and ((Get-Item -LiteralPath $solutionPath).Length -gt 100)
    Add-BooleanFeatureAssessment -Ctx $Ctx -Id 'engineering.solution_doc' -Level 'engineering' -Category 'documentation' -Requirement 'Non-empty docs/reshenie.md exists' -Implemented $solutionOk -Conformant $solutionOk -Evidence @('repo_snapshot/docs/reshenie.md')
}

function Copy-CheckPath {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Source,
        [Parameter(Mandatory=$true)][string]$RelativeDestination
    )

    if (-not (Test-Path -LiteralPath $Source)) {
        return
    }

    $destination = Join-Path $Ctx.ResultDir $RelativeDestination
    $parent = Split-Path -Parent $destination
    if ($parent) {
        New-Item -ItemType Directory -Force -Path $parent | Out-Null
    }
    Copy-Item -LiteralPath $Source -Destination $destination -Recurse -Force
}

function Complete-Check {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [hashtable]$Extra = @{}
    )

    Add-StandardEngineeringAssessments -Ctx $Ctx

    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_git_head' -Command "git rev-parse HEAD | Set-Content -LiteralPath '$($Ctx.MetaDir)\git_head.txt' -Encoding UTF8" | Out-Null
    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_git_status' -Command "`$statusPath = '$($Ctx.MetaDir)\git_status_short.txt'; `$status = git status --short; if (`$LASTEXITCODE -ne 0) { exit `$LASTEXITCODE }; if (`$null -eq `$status) { '' | Set-Content -LiteralPath `$statusPath -Encoding UTF8 } else { @(`$status) | Set-Content -LiteralPath `$statusPath -Encoding UTF8 }" | Out-Null
    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_go_version' -Command "& '$($Ctx.GoCmd)' version | Set-Content -LiteralPath '$($Ctx.MetaDir)\go_version.txt' -Encoding UTF8" | Out-Null
    Invoke-CheckCommand -Ctx $Ctx -Name 'meta_go_env' -Command "& '$($Ctx.GoCmd)' env GOVERSION GOOS GOARCH | Set-Content -LiteralPath '$($Ctx.MetaDir)\go_env.txt' -Encoding UTF8" | Out-Null

    foreach ($name in @('README.md', 'Makefile', 'go.mod', 'docs')) {
        $path = Join-Path $Ctx.RepoRoot $name
        Copy-CheckPath -Ctx $Ctx -Source $path -RelativeDestination "repo_snapshot/$name"
    }

    $assessmentItems = @($Ctx.Assessments)
    $assessmentSummary = [ordered]@{}
    foreach ($level in @('minimum','good','excellent','engineering')) {
        $items = @($assessmentItems | Where-Object { $_.level -eq $level })
        $assessmentSummary[$level] = [ordered]@{
            total = $items.Count
            full = @($items | Where-Object { $_.implementation -eq 'full' }).Count
            partial = @($items | Where-Object { $_.implementation -eq 'partial' }).Count
            not_implemented = @($items | Where-Object { $_.implementation -eq 'not_implemented' }).Count
            conformant = @($items | Where-Object { $_.conformance -eq 'conformant' }).Count
            nonconformant = @($items | Where-Object { $_.conformance -eq 'nonconformant' }).Count
            not_tested = @($items | Where-Object { $_.conformance -eq 'not_tested' }).Count
        }
    }
    Save-CheckJson -Path (Join-Path $Ctx.ResultDir 'assessment.json') -Value ([ordered]@{
        schema_version = 1
        statuses = [ordered]@{
            implementation = @('not_implemented','partial','full')
            conformance = @('not_tested','nonconformant','conformant')
        }
        summary = $assessmentSummary
        features = $assessmentItems
    })

    $manifest = [ordered]@{
        student = $Ctx.Student
        repo_root = $Ctx.RepoRoot
        started_at = $Ctx.StartedAt
        completed_at = (Get-Date).ToString('o')
        machine = [ordered]@{
            computer_name = $env:COMPUTERNAME
            user_name = $env:USERNAME
            os = (Get-CimInstance Win32_OperatingSystem).Caption
            powershell = $PSVersionTable.PSVersion.ToString()
        }
        result_dir = $Ctx.ResultDir
        commands_file = 'commands.jsonl'
        assessment_file = 'assessment.json'
        notes = $Extra
    }
    Save-CheckJson -Path (Join-Path $Ctx.ResultDir 'manifest.json') -Value $manifest

    $zipPath = "$($Ctx.ResultDir).zip"
    if (Test-Path -LiteralPath $zipPath) {
        Remove-Item -LiteralPath $zipPath -Force
    }
    Compress-Archive -Path (Join-Path $Ctx.ResultDir '*') -DestinationPath $zipPath -Force

    Write-Host "CHECK_RESULT_DIR=$($Ctx.ResultDir)"
    Write-Host "CHECK_RESULT_ZIP=$zipPath"
    return $zipPath
}


$ctx = New-CheckContext -Student 'reverse_index_query_check' -RepoRoot $RepoRoot -OutRoot $OutRoot

$eventsPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/events.jsonl' -Content @'
{"id":1,"timestamp":"2026-06-16T10:00:00Z","user_id":"user_001","department":"sales","action":"open_file","channel":"local","file_ext":"xlsx","destination_type":"none","severity":"low"}
{"id":2,"timestamp":"2026-06-16T10:05:00Z","user_id":"user_001","department":"sales","action":"email_send","channel":"email","file_ext":"xlsx","destination_type":"external","severity":"high"}
{"id":3,"timestamp":"2026-06-16T10:07:00Z","user_id":"user_002","department":"hr","action":"email_send","channel":"email","file_ext":"pdf","destination_type":"internal","severity":"medium"}
{"id":4,"timestamp":"2026-06-16T10:09:00Z","user_id":"user_003","department":"sales","action":"cloud_upload","channel":"cloud","file_ext":"zip","destination_type":"external","severity":"critical"}
'@

$queryPath = Write-CheckText -Ctx $ctx -RelativePath 'inputs/query.json' -Content @'
{
  "op": "AND",
  "left": {"op": "TERM", "field": "channel", "value": "email"},
  "right": {"op": "TERM", "field": "destination_type", "value": "external"}
}
'@

Invoke-CheckCommand -Ctx $ctx -Name 'go_test_all' -Command "& '$($ctx.GoCmd)' test ./..."

if (Test-Path -LiteralPath (Join-Path $ctx.RepoRoot 'Makefile')) {
    Invoke-CheckCommand -Ctx $ctx -Name 'make_test' -Command 'make test'
    Invoke-CheckCommand -Ctx $ctx -Name 'make_bench' -Command 'make bench'
    Invoke-CheckCommand -Ctx $ctx -Name 'make_demo' -Command 'make demo'
}

$tool = Join-Path $ctx.OutputsDir 'reverse-index-query.exe'
Invoke-CheckCommand -Ctx $ctx -Name 'build_cli' -Command "& '$($ctx.GoCmd)' build -o '$tool' ./cmd/reverse-index-query"

$generated = Join-Path $ctx.OutputsDir 'generated_events.jsonl'
$scan = Join-Path $ctx.OutputsDir 'scan.json'
$index = Join-Path $ctx.OutputsDir 'index.json'
$compare = Join-Path $ctx.OutputsDir 'compare.md'

Invoke-CheckCommand -Ctx $ctx -Name 'cli_generate' -Command "& '$tool' generate --count 25 --out '$generated' --seed 42"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_run_scan' -Command "& '$tool' run --events '$eventsPath' --query '$queryPath' --method scan --out '$scan'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_run_index' -Command "& '$tool' run --events '$eventsPath' --query '$queryPath' --method index --out '$index'"
Invoke-CheckCommand -Ctx $ctx -Name 'cli_compare' -Command "& '$tool' compare --events '$eventsPath' --query '$queryPath' --out '$compare'"

$comparisonPath = Join-Path $ctx.OutputsDir 'scan_index_comparison.json'
$comparison = [ordered]@{
    scan_exists = Test-Path -LiteralPath $scan
    index_exists = Test-Path -LiteralPath $index
    matched_ids_equal = $false
}
try {
    if ($comparison.scan_exists -and $comparison.index_exists) {
        $scanJson = Get-Content -LiteralPath $scan -Raw | ConvertFrom-Json
        $indexJson = Get-Content -LiteralPath $index -Raw | ConvertFrom-Json
        $comparison.matched_ids_equal = (($scanJson.matched_ids -join ',') -eq ($indexJson.matched_ids -join ','))
        $comparison.scan_matched_count = $scanJson.matched_count
        $comparison.index_matched_count = $indexJson.matched_count
    }
} catch {
    $comparison.error = $_.Exception.Message
}
Save-CheckJson -Path $comparisonPath -Value $comparison

$expectedArtifacts = [ordered]@{
    generated_events = Test-Path -LiteralPath $generated
    scan_json = Test-Path -LiteralPath $scan
    index_json = Test-Path -LiteralPath $index
    compare_report = Test-Path -LiteralPath $compare
    comparison = Test-Path -LiteralPath $comparisonPath
}
Save-CheckJson -Path (Join-Path $ctx.OutputsDir 'artifact_presence.json') -Value $expectedArtifacts

Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.generator' -Level 'minimum' -Category 'cli' -Requirement 'JSONL event generator' -CommandName 'cli_generate' -RequiredArtifacts @($generated)
Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.jsonl_reader' -Level 'minimum' -Category 'input' -Requirement 'Read JSONL events' -CommandName 'cli_run_scan' -RequiredArtifacts @($scan)
Add-CommandFeatureAssessment -Ctx $ctx -Id 'minimum.reverse_index' -Level 'minimum' -Category 'algorithm' -Requirement 'Reverse index build' -CommandName 'cli_run_index' -RequiredArtifacts @($index)
Add-BooleanFeatureAssessment -Ctx $ctx -Id 'minimum.scan_equals_index' -Level 'minimum' -Category 'correctness' -Requirement 'Scan and index return same IDs' -Implemented ($comparison.scan_exists -and $comparison.index_exists) -Conformant ([bool]$comparison.matched_ids_equal) -Evidence @('outputs/scan.json','outputs/index.json','outputs/scan_index_comparison.json')

# Runtime check for TERM/AND/OR/nested query tree on fixed four events.
$queryTreeCases = @(
    [ordered]@{
        name = 'term_email'
        expected_ids = @([UInt64]2, [UInt64]3)
        query_object = [ordered]@{ op = 'TERM'; field = 'channel'; value = 'email' }
    },
    [ordered]@{
        name = 'and_email_external'
        expected_ids = @([UInt64]2)
        query_object = [ordered]@{
            op = 'AND'
            left = [ordered]@{ op = 'TERM'; field = 'channel'; value = 'email' }
            right = [ordered]@{ op = 'TERM'; field = 'destination_type'; value = 'external' }
        }
    },
    [ordered]@{
        name = 'or_email_external'
        expected_ids = @([UInt64]2, [UInt64]3, [UInt64]4)
        query_object = [ordered]@{
            op = 'OR'
            left = [ordered]@{ op = 'TERM'; field = 'channel'; value = 'email' }
            right = [ordered]@{ op = 'TERM'; field = 'destination_type'; value = 'external' }
        }
    },
    [ordered]@{
        name = 'nested_and_or'
        expected_ids = @([UInt64]2, [UInt64]4)
        query_object = [ordered]@{
            op = 'AND'
            left = [ordered]@{
                op = 'OR'
                left = [ordered]@{ op = 'TERM'; field = 'channel'; value = 'email' }
                right = [ordered]@{ op = 'TERM'; field = 'destination_type'; value = 'external' }
            }
            right = [ordered]@{ op = 'TERM'; field = 'department'; value = 'sales' }
        }
    }
)
$queryTreeResults = @()
$queryTreeTempFiles = @()
$queryTreeAnyExecuted = $false
$queryTreeAllExecuted = $true
$queryTreeConformant = $true
foreach ($case in $queryTreeCases) {
    $caseQueryPath = Join-Path $ctx.TmpDir "query_tree_$($case.name).json"
    $queryTreeTempFiles += $caseQueryPath
    Set-Content -LiteralPath $caseQueryPath -Value (($case.query_object | ConvertTo-Json -Depth 20) + "`n") -Encoding UTF8
    $caseScanPath = Join-Path $ctx.OutputsDir "query_tree_$($case.name)_scan.json"
    $caseIndexPath = Join-Path $ctx.OutputsDir "query_tree_$($case.name)_index.json"
    $scanCommandName = "cli_query_tree_$($case.name)_scan"
    $indexCommandName = "cli_query_tree_$($case.name)_index"
    Invoke-CheckCommand -Ctx $ctx -Name $scanCommandName -Command "& '$tool' run --events '$eventsPath' --query '$caseQueryPath' --method scan --out '$caseScanPath'"
    Invoke-CheckCommand -Ctx $ctx -Name $indexCommandName -Command "& '$tool' run --events '$eventsPath' --query '$caseQueryPath' --method index --out '$caseIndexPath'"

    $scanExit = if ($ctx.CommandResults.ContainsKey($scanCommandName)) { [int]$ctx.CommandResults[$scanCommandName].exit_code } else { -999 }
    $indexExit = if ($ctx.CommandResults.ContainsKey($indexCommandName)) { [int]$ctx.CommandResults[$indexCommandName].exit_code } else { -999 }
    if ($scanExit -ne -999 -or $indexExit -ne -999) {
        $queryTreeAnyExecuted = $true
    }
    $scanSummary = Get-RunResultSummary -Path $caseScanPath
    $indexSummary = Get-RunResultSummary -Path $caseIndexPath
    $expectedIds = @([UInt64[]]$case.expected_ids | Sort-Object)
    $scanIdsOk = Test-UInt64ArrayEqual -Left $scanSummary.matched_ids -Right $expectedIds
    $indexIdsOk = Test-UInt64ArrayEqual -Left $indexSummary.matched_ids -Right $expectedIds
    $methodsEqual = Test-UInt64ArrayEqual -Left $scanSummary.matched_ids -Right $indexSummary.matched_ids
    $scanMethodOk = ([string]$scanSummary.method).ToLowerInvariant() -eq 'scan'
    $indexMethodOk = ([string]$indexSummary.method).ToLowerInvariant() -eq 'index'
    $caseConformant = ($scanExit -eq 0) -and ($indexExit -eq 0) -and ($scanSummary.parse_error -eq '') -and ($indexSummary.parse_error -eq '') -and $scanIdsOk -and $indexIdsOk -and $methodsEqual -and $scanMethodOk -and $indexMethodOk
    if (-not $caseConformant) {
        $queryTreeConformant = $false
    }
    if (($scanExit -ne 0) -or ($indexExit -ne 0) -or (-not $scanSummary.exists) -or (-not $indexSummary.exists)) {
        $queryTreeAllExecuted = $false
    }
    $queryTreeResults += [ordered]@{
        name = $case.name
        expected_ids = $expectedIds
        scan_exit_code = $scanExit
        index_exit_code = $indexExit
        scan_method = $scanSummary.method
        index_method = $indexSummary.method
        scan_ids = $scanSummary.matched_ids
        index_ids = $indexSummary.matched_ids
        scan_parse_error = $scanSummary.parse_error
        index_parse_error = $indexSummary.parse_error
        conformant = $caseConformant
    }
}
$queryTreeEvidencePath = Join-Path $ctx.OutputsDir 'query_tree_runtime_evidence.json'
Save-CheckJson -Path $queryTreeEvidencePath -Value ([ordered]@{
    cases = $queryTreeResults
    all_executed = $queryTreeAllExecuted
    conformant = $queryTreeConformant
})
$queryTreeImplementation = if ($queryTreeAllExecuted) { 'full' } elseif ($queryTreeAnyExecuted) { 'partial' } else { 'not_implemented' }
$queryTreeConformance = if ($queryTreeConformant -and $queryTreeAllExecuted) { 'conformant' } else { 'nonconformant' }
$queryTreeFailed = @($queryTreeResults | Where-Object { -not $_.conformant } | ForEach-Object { $_.name })
$queryTreeDetails = if ($queryTreeFailed.Count -eq 0) { 'all runtime query cases passed with exact IDs and methods' } else { "failed runtime cases: $($queryTreeFailed -join ', ')" }

# Targeted go test -json probes for Intersect and Union/Or.
$intersectGoTestJsonPath = Join-Path $ctx.OutputsDir 'go_test_intersect.jsonl'
$unionOrGoTestJsonPath = Join-Path $ctx.OutputsDir 'go_test_union_or.jsonl'
Invoke-CheckCommand -Ctx $ctx -Name 'go_test_intersect_json' -Command "& '$($ctx.GoCmd)' test -json ./... -run 'Intersect' | Tee-Object -FilePath '$intersectGoTestJsonPath'"
Invoke-CheckCommand -Ctx $ctx -Name 'go_test_union_or_json' -Command "& '$($ctx.GoCmd)' test -json ./... -run 'Union|Or' | Tee-Object -FilePath '$unionOrGoTestJsonPath'"
$intersectGoTestSummary = Get-GoTestJsonSummary -Path $intersectGoTestJsonPath -TestNameRegex 'Intersect'
$unionOrGoTestSummary = Get-GoTestJsonSummary -Path $unionOrGoTestJsonPath -TestNameRegex 'Union|Or'
$listTestsEvidencePath = Join-Path $ctx.OutputsDir 'list_tests_runtime_evidence.json'
Save-CheckJson -Path $listTestsEvidencePath -Value ([ordered]@{
    intersect = $intersectGoTestSummary
    union_or = $unionOrGoTestSummary
})
$listTestsImplementation = if ($intersectGoTestSummary.exists -and $unionOrGoTestSummary.exists) {
    if ($intersectGoTestSummary.matching_passed -and $unionOrGoTestSummary.matching_passed) { 'full' } else { 'partial' }
} else {
    'not_implemented'
}
$listTestsConformance = if ($intersectGoTestSummary.matching_passed -and $unionOrGoTestSummary.matching_passed) { 'conformant' } else { 'nonconformant' }
$listTestsDetails = "intersect_passed=$($intersectGoTestSummary.matching_passed); union_or_passed=$($unionOrGoTestSummary.matching_passed); intersect_tests=$($intersectGoTestSummary.passed_tests -join ','); union_or_tests=$($unionOrGoTestSummary.passed_tests -join ',')"

# Runtime 1M scenario with preflight/timeout and compact evidence.
$millionEventsPath = Join-Path $ctx.OutputsDir 'events_1000000.jsonl'
$millionQueryPath = Join-Path $ctx.TmpDir 'query_1000000.json'
$millionScanPath = Join-Path $ctx.OutputsDir 'million_scan.json'
$millionIndexPath = Join-Path $ctx.OutputsDir 'million_index.json'
$millionComparePath = Join-Path $ctx.OutputsDir 'million_compare.md'
$queryTreeTempFiles += $millionQueryPath
Set-Content -LiteralPath $millionQueryPath -Value (@'
{
  "op": "TERM",
  "field": "channel",
  "value": "__never__"
}
'@) -Encoding UTF8

$outputsQualifier = Split-Path -Path $ctx.OutputsDir -Qualifier
$outputsDriveName = $outputsQualifier.TrimEnd('\').TrimEnd(':')
$outputsDrive = if ($outputsDriveName -ne '') { Get-PSDrive -Name $outputsDriveName -ErrorAction SilentlyContinue } else { $null }
$millionPreflightMinFreeBytes = [Int64](256MB)
$millionPreflightOk = $true
if ($null -ne $outputsDrive -and [Int64]$outputsDrive.Free -lt $millionPreflightMinFreeBytes) {
    $millionPreflightOk = $false
}

$millionRuntime = [ordered]@{
    preflight_ok = $millionPreflightOk
    preflight_min_free_bytes = $millionPreflightMinFreeBytes
    drive_free_bytes = if ($null -ne $outputsDrive) { [Int64]$outputsDrive.Free } else { $null }
    generated_exists = $false
    line_count = 0
    size_bytes = 0
    generate_duration_ms = $null
    scan_duration_ms = $null
    index_duration_ms = $null
    compare_duration_ms = $null
    scan_exit_code = $null
    index_exit_code = $null
    compare_exit_code = $null
    ids_equal = $false
    report_mentions_equality = $false
    scan_ids = @()
    index_ids = @()
    index_peak_working_set_bytes = $null
}

if ($millionPreflightOk) {
    Invoke-CheckCommand -Ctx $ctx -Name 'cli_generate_million' -Command "& '$tool' generate --count 1000000 --out '$millionEventsPath' --seed 424242"
    if ($ctx.CommandResults.ContainsKey('cli_generate_million')) {
        $millionRuntime.generate_duration_ms = [int]$ctx.CommandResults['cli_generate_million'].duration_ms
    }
    $millionRuntime.generated_exists = Test-Path -LiteralPath $millionEventsPath
    if ($millionRuntime.generated_exists) {
        $lineCount = 0
        Get-Content -LiteralPath $millionEventsPath -ReadCount 5000 | ForEach-Object { $lineCount += $_.Count }
        $millionRuntime.line_count = [int64]$lineCount
        $millionRuntime.size_bytes = [Int64](Get-Item -LiteralPath $millionEventsPath).Length
    }

    Invoke-CheckNativeCommand -Ctx $ctx -Name 'cli_run_index_million' -FilePath $tool -ArgumentList @('run', '--events', $millionEventsPath, '--query', $millionQueryPath, '--method', 'index', '--out', $millionIndexPath) -TimeoutSec 600 -CapturePeakWorkingSet $true
    Invoke-CheckNativeCommand -Ctx $ctx -Name 'cli_run_scan_million' -FilePath $tool -ArgumentList @('run', '--events', $millionEventsPath, '--query', $millionQueryPath, '--method', 'scan', '--out', $millionScanPath) -TimeoutSec 600
    Invoke-CheckNativeCommand -Ctx $ctx -Name 'cli_compare_million' -FilePath $tool -ArgumentList @('compare', '--events', $millionEventsPath, '--query', $millionQueryPath, '--out', $millionComparePath) -TimeoutSec 600

    if ($ctx.CommandResults.ContainsKey('cli_run_index_million')) {
        $millionRuntime.index_duration_ms = [int]$ctx.CommandResults['cli_run_index_million'].duration_ms
        $millionRuntime.index_exit_code = [int]$ctx.CommandResults['cli_run_index_million'].exit_code
        if ($ctx.CommandResults['cli_run_index_million'].Contains('peak_working_set_bytes')) {
            $millionRuntime.index_peak_working_set_bytes = [Int64]$ctx.CommandResults['cli_run_index_million'].peak_working_set_bytes
        }
    }
    if ($ctx.CommandResults.ContainsKey('cli_run_scan_million')) {
        $millionRuntime.scan_duration_ms = [int]$ctx.CommandResults['cli_run_scan_million'].duration_ms
        $millionRuntime.scan_exit_code = [int]$ctx.CommandResults['cli_run_scan_million'].exit_code
    }
    if ($ctx.CommandResults.ContainsKey('cli_compare_million')) {
        $millionRuntime.compare_duration_ms = [int]$ctx.CommandResults['cli_compare_million'].duration_ms
        $millionRuntime.compare_exit_code = [int]$ctx.CommandResults['cli_compare_million'].exit_code
    }

    $millionScanSummary = Get-RunResultSummary -Path $millionScanPath
    $millionIndexSummary = Get-RunResultSummary -Path $millionIndexPath
    $millionRuntime.scan_ids = $millionScanSummary.matched_ids
    $millionRuntime.index_ids = $millionIndexSummary.matched_ids
    $millionRuntime.ids_equal = Test-UInt64ArrayEqual -Left $millionScanSummary.matched_ids -Right $millionIndexSummary.matched_ids
    if (Test-Path -LiteralPath $millionComparePath) {
        $millionCompareText = Get-Content -LiteralPath $millionComparePath -Raw -Encoding UTF8
        $millionRuntime.report_mentions_equality = $millionCompareText -match '(?is)(equal|same|совпад)'
    }
}
$millionEvidencePath = Join-Path $ctx.OutputsDir 'million_runtime_evidence.json'
Save-CheckJson -Path $millionEvidencePath -Value $millionRuntime

$millionHasComparison = (($millionRuntime.scan_exit_code -eq 0) -and ($millionRuntime.index_exit_code -eq 0) -and $millionRuntime.ids_equal) -or (($millionRuntime.compare_exit_code -eq 0) -and $millionRuntime.report_mentions_equality)
$millionImplementation = if ($millionRuntime.preflight_ok -and $millionRuntime.generated_exists -and (($millionRuntime.scan_exit_code -ne $null) -or ($millionRuntime.compare_exit_code -ne $null))) { 'full' } elseif ($millionRuntime.generated_exists) { 'partial' } else { 'not_implemented' }
$millionConformance = if ($millionRuntime.preflight_ok -and ($millionRuntime.line_count -eq 1000000) -and $millionHasComparison) { 'conformant' } else { 'nonconformant' }
$millionDetails = "preflight_ok=$($millionRuntime.preflight_ok); line_count=$($millionRuntime.line_count); size_bytes=$($millionRuntime.size_bytes); ids_equal=$($millionRuntime.ids_equal); report_mentions_equality=$($millionRuntime.report_mentions_equality)"

# Index build duration metric must be explicit in runtime output/report.
$indexBuildDurationMs = $null
$indexBuildDurationSource = ''
if (Test-Path -LiteralPath $millionIndexPath) {
    try {
        $millionIndexJson = Get-Content -LiteralPath $millionIndexPath -Raw -Encoding UTF8 | ConvertFrom-Json
        $indexBuildDurationMs = Find-NumericFieldRecursive -Node $millionIndexJson -FieldNames @('index_build_duration_ms', 'index_build_ms', 'index_build_duration', 'indexBuildDurationMs')
        if ($null -ne $indexBuildDurationMs) {
            $indexBuildDurationSource = 'outputs/million_index.json'
        }
    } catch {
    }
}
if ($null -eq $indexBuildDurationMs -and (Test-Path -LiteralPath $millionComparePath)) {
    $millionCompareText = Get-Content -LiteralPath $millionComparePath -Raw -Encoding UTF8
    $buildDurationMatch = [regex]::Match($millionCompareText, '(?im)(index[_\s-]*build[_\s-]*duration(?:_ms)?|build[_\s-]*duration(?:_ms)?)\s*[:=|]\s*([0-9]+(?:\.[0-9]+)?)')
    if ($buildDurationMatch.Success) {
        try {
            $indexBuildDurationMs = [double]$buildDurationMatch.Groups[2].Value
            $indexBuildDurationSource = 'outputs/million_compare.md'
        } catch {
            $indexBuildDurationMs = $null
        }
    }
}
$indexBuildDurationEvidencePath = Join-Path $ctx.OutputsDir 'index_build_duration_evidence.json'
$indexBuildDurationLimitMs = [double]10000
Save-CheckJson -Path $indexBuildDurationEvidencePath -Value ([ordered]@{
    million_runtime_executed = $millionRuntime.preflight_ok
    metric_found = ($null -ne $indexBuildDurationMs)
    metric_value_ms = $indexBuildDurationMs
    limit_ms = $indexBuildDurationLimitMs
    metric_source = $indexBuildDurationSource
    expected_fields = @('index_build_duration_ms', 'index_build_ms', 'index_build_duration', 'indexBuildDurationMs')
})
$indexBuildImplementation = if ($null -ne $indexBuildDurationMs) { 'full' } else { 'not_implemented' }
$indexBuildConformance = if (($null -ne $indexBuildDurationMs) -and ([double]$indexBuildDurationMs -gt 0) -and ([double]$indexBuildDurationMs -le $indexBuildDurationLimitMs)) { 'conformant' } else { 'nonconformant' }
$indexBuildDetails = if ($null -ne $indexBuildDurationMs) { "index build duration metric found: $indexBuildDurationMs ms; limit=$indexBuildDurationLimitMs ms ($indexBuildDurationSource)" } else { 'runtime on 1M executed, but explicit index build duration metric not found in output/report' }

# Probe for string query support: documented flag when available, fallback to safe expected-fail probe.
$stringQueryScanPath = Join-Path $ctx.OutputsDir 'string_query_scan.json'
$stringQueryIndexPath = Join-Path $ctx.OutputsDir 'string_query_index.json'
$stringQueryProbePath = Join-Path $ctx.TmpDir 'string_query_probe.txt'
$queryTreeTempFiles += $stringQueryProbePath
$stringQueryEvidencePath = Join-Path $ctx.OutputsDir 'string_query_evidence.json'
$stringQueryExpression = 'channel=email AND destination_type=external'
$stringQueryFlag = ''
$stringQueryMode = 'fallback_query_file'
$stringQueryImplementation = 'not_implemented'
$stringQueryConformance = 'nonconformant'
$stringQueryDetails = 'string query support was not detected'
Invoke-CheckCommand -Ctx $ctx -Name 'cli_run_help' -Command "& '$tool' run --help"
$runHelpText = ''
if ($ctx.CommandResults.ContainsKey('cli_run_help')) {
    $runHelpLogRelative = [string]$ctx.CommandResults['cli_run_help'].log
    $runHelpLogPath = Join-Path $ctx.ResultDir $runHelpLogRelative.Replace('/', '\')
    if (Test-Path -LiteralPath $runHelpLogPath) {
        $runHelpText = Get-Content -LiteralPath $runHelpLogPath -Raw -Encoding UTF8
    }
}
foreach ($candidate in @('--query-string', '--query-str', '--query-text', '--query-expression', '--expr')) {
    if ($runHelpText -match [regex]::Escape($candidate)) {
        $stringQueryFlag = $candidate
        break
    }
}
if ($stringQueryFlag -ne '') {
    $stringQueryMode = 'help_flag'
    Invoke-CheckNativeCommand -Ctx $ctx -Name 'cli_run_string_query_scan' -FilePath $tool -ArgumentList @('run', '--events', $eventsPath, $stringQueryFlag, $stringQueryExpression, '--method', 'scan', '--out', $stringQueryScanPath) -TimeoutSec 120
    Invoke-CheckNativeCommand -Ctx $ctx -Name 'cli_run_string_query_index' -FilePath $tool -ArgumentList @('run', '--events', $eventsPath, $stringQueryFlag, $stringQueryExpression, '--method', 'index', '--out', $stringQueryIndexPath) -TimeoutSec 120
    $stringScanSummary = Get-RunResultSummary -Path $stringQueryScanPath
    $stringIndexSummary = Get-RunResultSummary -Path $stringQueryIndexPath
    $stringExpectedIds = @([UInt64]2)
    $scanOk = ($ctx.CommandResults['cli_run_string_query_scan'].exit_code -eq 0) -and (([string]$stringScanSummary.method).ToLowerInvariant() -eq 'scan') -and (Test-UInt64ArrayEqual -Left $stringScanSummary.matched_ids -Right $stringExpectedIds)
    $indexOk = ($ctx.CommandResults['cli_run_string_query_index'].exit_code -eq 0) -and (([string]$stringIndexSummary.method).ToLowerInvariant() -eq 'index') -and (Test-UInt64ArrayEqual -Left $stringIndexSummary.matched_ids -Right $stringExpectedIds)
    if ($scanOk -and $indexOk) {
        $stringQueryImplementation = 'full'
        $stringQueryConformance = 'conformant'
        $stringQueryDetails = "string query is supported via $stringQueryFlag and returned exact expected IDs"
    } else {
        $stringQueryImplementation = 'partial'
        $stringQueryConformance = 'nonconformant'
        $stringQueryDetails = "string query probe via $stringQueryFlag did not return exact expected IDs"
    }
} else {
    Set-Content -LiteralPath $stringQueryProbePath -Value "$stringQueryExpression`n" -Encoding UTF8
    Invoke-CheckCommand -Ctx $ctx -Name 'cli_run_string_query_probe' -Command "& '$tool' run --events '$eventsPath' --query '$stringQueryProbePath' --method scan --out '$stringQueryScanPath'"
    $stringProbeExit = if ($ctx.CommandResults.ContainsKey('cli_run_string_query_probe')) { [int]$ctx.CommandResults['cli_run_string_query_probe'].exit_code } else { -999 }
    $stringProbeSummary = Get-RunResultSummary -Path $stringQueryScanPath
    if ($stringProbeExit -eq 0) {
        $stringExpectedIds = @([UInt64]2)
        $probeExact = (([string]$stringProbeSummary.method).ToLowerInvariant() -eq 'scan') -and (Test-UInt64ArrayEqual -Left $stringProbeSummary.matched_ids -Right $stringExpectedIds)
        if ($probeExact) {
            $stringQueryImplementation = 'full'
            $stringQueryConformance = 'conformant'
            $stringQueryDetails = 'string query accepted through query file and returned exact expected IDs'
        } else {
            $stringQueryImplementation = 'partial'
            $stringQueryConformance = 'nonconformant'
            $stringQueryDetails = 'query file string probe succeeded, but IDs/method did not match expected runtime result'
        }
    } else {
        $stringQueryImplementation = 'not_implemented'
        $stringQueryConformance = 'nonconformant'
        $stringQueryDetails = 'no documented string-query flag found; fallback probe with string query file failed as expected'
    }
}
Save-CheckJson -Path $stringQueryEvidencePath -Value ([ordered]@{
    mode = $stringQueryMode
    detected_flag = $stringQueryFlag
    expected_ids = @([UInt64]2)
    implementation = $stringQueryImplementation
    conformance = $stringQueryConformance
    details = $stringQueryDetails
})

# Runtime nested AND scenario with skewed posting sizes.
$intersectionEventsPath = Join-Path $ctx.OutputsDir 'intersection_order_events.jsonl'
$intersectionQueryPath = Join-Path $ctx.TmpDir 'intersection_order_query.json'
$intersectionOutputPath = Join-Path $ctx.OutputsDir 'intersection_order_index.json'
$queryTreeTempFiles += $intersectionQueryPath
$intersectionLines = New-Object System.Collections.Generic.List[string]
for ($i = 1; $i -le 120; $i++) {
    $channel = if ($i -le 100) { 'email' } else { 'cloud' }
    $destination = if ($i -le 30) { 'external' } else { 'internal' }
    $department = if ($i -le 10) { 'sales' } else { 'hr' }
    $intersectionLines.Add((([ordered]@{
        id = $i
        channel = $channel
        destination_type = $destination
        department = $department
    } | ConvertTo-Json -Compress))) | Out-Null
}
Set-Content -LiteralPath $intersectionEventsPath -Value $intersectionLines -Encoding UTF8
Set-Content -LiteralPath $intersectionQueryPath -Value (([ordered]@{
    op = 'AND'
    left = [ordered]@{
        op = 'AND'
        left = [ordered]@{ op = 'TERM'; field = 'channel'; value = 'email' }
        right = [ordered]@{ op = 'TERM'; field = 'destination_type'; value = 'external' }
    }
    right = [ordered]@{ op = 'TERM'; field = 'department'; value = 'sales' }
} | ConvertTo-Json -Depth 20) + "`n") -Encoding UTF8
Invoke-CheckCommand -Ctx $ctx -Name 'cli_run_intersection_order' -Command "& '$tool' run --events '$intersectionEventsPath' --query '$intersectionQueryPath' --method index --out '$intersectionOutputPath'"
$intersectionSummary = Get-RunResultSummary -Path $intersectionOutputPath
$intersectionExpectedIds = @()
for ($id = 1; $id -le 10; $id++) {
    $intersectionExpectedIds += [UInt64]$id
}
$intersectionIdsOk = Test-UInt64ArrayEqual -Left $intersectionSummary.matched_ids -Right $intersectionExpectedIds
$intersectionObservableOrder = @()
$intersectionHasTrace = $false
$intersectionOrderConformant = $false
if (Test-Path -LiteralPath $intersectionOutputPath) {
    try {
        $intersectionJson = Get-Content -LiteralPath $intersectionOutputPath -Raw -Encoding UTF8 | ConvertFrom-Json
        if ($intersectionJson.PSObject.Properties.Name -contains 'intersection_order') {
            $intersectionObservableOrder = @($intersectionJson.intersection_order | ForEach-Object { [string]$_ })
        } elseif ($intersectionJson.PSObject.Properties.Name -contains 'plan' -and $null -ne $intersectionJson.plan -and $intersectionJson.plan.PSObject.Properties.Name -contains 'intersection_order') {
            $intersectionObservableOrder = @($intersectionJson.plan.intersection_order | ForEach-Object { [string]$_ })
        } elseif ($intersectionJson.PSObject.Properties.Name -contains 'trace' -and $null -ne $intersectionJson.trace -and $intersectionJson.trace.PSObject.Properties.Name -contains 'intersection_order') {
            $intersectionObservableOrder = @($intersectionJson.trace.intersection_order | ForEach-Object { [string]$_ })
        }
    } catch {
    }
}
$intersectionHasTrace = ($intersectionObservableOrder.Count -gt 0)
if ($intersectionHasTrace) {
    $normalizedOrder = @($intersectionObservableOrder | ForEach-Object { $_.ToLowerInvariant() })
    $intersectionOrderConformant = ($normalizedOrder.Count -ge 3) -and ($normalizedOrder[0] -match 'department') -and ($normalizedOrder[1] -match 'destination') -and ($normalizedOrder[2] -match 'channel')
}
$intersectionImplementation = if ($ctx.CommandResults.ContainsKey('cli_run_intersection_order') -and [int]$ctx.CommandResults['cli_run_intersection_order'].exit_code -eq 0 -and (Test-Path -LiteralPath $intersectionOutputPath)) {
    if ($intersectionHasTrace) { 'full' } else { 'partial' }
} elseif ($ctx.CommandResults.ContainsKey('cli_run_intersection_order')) {
    'partial'
} else {
    'not_implemented'
}
$intersectionConformance = if ($intersectionIdsOk -and $intersectionOrderConformant) { 'conformant' } else { 'nonconformant' }
$intersectionDetails = if ($intersectionHasTrace) { "observed_order=$($intersectionObservableOrder -join ' -> '); ids_ok=$intersectionIdsOk" } else { "nested AND executed with skewed posting sizes, but CLI output has no observable plan/trace order; ids_ok=$intersectionIdsOk" }
$intersectionEvidencePath = Join-Path $ctx.OutputsDir 'intersection_order_evidence.json'
Save-CheckJson -Path $intersectionEvidencePath -Value ([ordered]@{
    expected_ids = $intersectionExpectedIds
    actual_ids = $intersectionSummary.matched_ids
    ids_ok = $intersectionIdsOk
    observable_order = $intersectionObservableOrder
    order_conformant = $intersectionOrderConformant
    implementation = $intersectionImplementation
    conformance = $intersectionConformance
})

# Memory estimate assessment based on 1M index run peak and optional explicit metric.
$memoryLimitBytes = [Int64](768MB)
$indexPeakWorkingSetBytes = if ($ctx.CommandResults.ContainsKey('cli_run_index_million') -and $ctx.CommandResults['cli_run_index_million'].Contains('peak_working_set_bytes')) { [Int64]$ctx.CommandResults['cli_run_index_million'].peak_working_set_bytes } else { [Int64]0 }
$reportedIndexMemoryBytes = $null
if (Test-Path -LiteralPath $millionIndexPath) {
    try {
        $millionIndexJson = Get-Content -LiteralPath $millionIndexPath -Raw -Encoding UTF8 | ConvertFrom-Json
        $reportedIndexMemoryBytes = Find-NumericFieldRecursive -Node $millionIndexJson -FieldNames @('index_memory_bytes', 'memory_estimate_bytes', 'index_estimated_bytes', 'index_size_bytes')
    } catch {
        $reportedIndexMemoryBytes = $null
    }
}
$memoryImplementation = 'not_implemented'
$memoryConformance = 'nonconformant'
$memoryDetails = 'memory runtime evidence was not produced'
if ($null -ne $reportedIndexMemoryBytes) {
    $memoryImplementation = 'full'
    if (([double]$reportedIndexMemoryBytes -le [double]$memoryLimitBytes) -and (($indexPeakWorkingSetBytes -le 0) -or ($indexPeakWorkingSetBytes -le $memoryLimitBytes))) {
        $memoryConformance = 'conformant'
    }
    $memoryDetails = "reported index memory metric found ($reportedIndexMemoryBytes bytes); peak_working_set_bytes=$indexPeakWorkingSetBytes; limit_bytes=$memoryLimitBytes"
} elseif ($indexPeakWorkingSetBytes -gt 0) {
    $memoryImplementation = 'partial'
    if ($indexPeakWorkingSetBytes -le $memoryLimitBytes) {
        $memoryConformance = 'conformant'
    }
    $memoryDetails = "no explicit index memory estimate found; used process PeakWorkingSet64=$indexPeakWorkingSetBytes bytes with limit=$memoryLimitBytes bytes"
}
$memoryEvidencePath = Join-Path $ctx.OutputsDir 'memory_runtime_evidence.json'
Save-CheckJson -Path $memoryEvidencePath -Value ([ordered]@{
    limit_bytes = $memoryLimitBytes
    peak_working_set_bytes = $indexPeakWorkingSetBytes
    reported_index_memory_bytes = $reportedIndexMemoryBytes
    implementation = $memoryImplementation
    conformance = $memoryConformance
})

# Documentation trade-off check from README/docs/reshenie.md.
$docsEvidencePath = Join-Path $ctx.OutputsDir 'index_tradeoffs_evidence.json'
$docsTradeoff = Get-DocumentationTradeoffEvidence -DocPaths @(
    (Join-Path $ctx.RepoRoot 'README.md'),
    (Join-Path $ctx.RepoRoot 'docs\reshenie.md')
)
$docsScore = 0
foreach ($flag in @('has_build_cost', 'has_memory_cost', 'has_repeat_query_benefit', 'has_large_data_benefit', 'mentions_scan_and_index')) {
    if ($docsTradeoff[$flag]) {
        $docsScore++
    }
}
$docsImplementation = 'not_implemented'
$docsConformance = 'nonconformant'
$docsDetails = 'README/docs evidence for index trade-offs was not found'
if ($docsTradeoff.existing_files.Count -gt 0) {
    if ($docsTradeoff.has_build_cost -and $docsTradeoff.has_memory_cost -and $docsTradeoff.has_repeat_query_benefit -and $docsTradeoff.has_large_data_benefit -and $docsTradeoff.mentions_scan_and_index) {
        $docsImplementation = 'full'
        $docsConformance = 'conformant'
        $docsDetails = 'documentation includes build and memory costs plus repeated-query and large-data benefits'
    } elseif (($docsTradeoff.has_build_cost -or $docsTradeoff.has_memory_cost) -and ($docsTradeoff.has_repeat_query_benefit -or $docsTradeoff.has_large_data_benefit) -and $docsTradeoff.mentions_scan_and_index) {
        $docsImplementation = 'partial'
        $docsConformance = 'conformant'
        $docsDetails = 'documentation partially covers trade-offs and benefits'
    } else {
        $docsImplementation = 'partial'
        $docsConformance = 'nonconformant'
        $docsDetails = 'documentation exists but does not sufficiently cover required index/scan trade-offs'
    }
}
Save-CheckJson -Path $docsEvidencePath -Value ([ordered]@{
    existing_files = $docsTradeoff.existing_files
    has_build_cost = $docsTradeoff.has_build_cost
    has_memory_cost = $docsTradeoff.has_memory_cost
    has_repeat_query_benefit = $docsTradeoff.has_repeat_query_benefit
    has_large_data_benefit = $docsTradeoff.has_large_data_benefit
    mentions_scan_and_index = $docsTradeoff.mentions_scan_and_index
    score = $docsScore
    implementation = $docsImplementation
    conformance = $docsConformance
})

# Remove large and temporary files before packaging results.
foreach ($cleanupPath in @($millionEventsPath, $stringQueryProbePath, $millionQueryPath, $intersectionQueryPath) + $queryTreeTempFiles) {
    if ($cleanupPath -and (Test-Path -LiteralPath $cleanupPath)) {
        Remove-Item -LiteralPath $cleanupPath -Force -ErrorAction SilentlyContinue
    }
}

Add-FeatureAssessment -Ctx $ctx -Id 'minimum.query_tree' -Level 'minimum' -Category 'algorithm' -Requirement 'JSON query tree TERM AND OR' -Implementation $queryTreeImplementation -Conformance $queryTreeConformance -Evidence @('outputs/query_tree_runtime_evidence.json') -Details $queryTreeDetails
Add-FeatureAssessment -Ctx $ctx -Id 'minimum.list_tests' -Level 'minimum' -Category 'tests' -Requirement 'Intersect and union unit tests exist' -Implementation $listTestsImplementation -Conformance $listTestsConformance -Evidence @('outputs/go_test_intersect.jsonl', 'outputs/go_test_union_or.jsonl', 'outputs/list_tests_runtime_evidence.json') -Details $listTestsDetails

Add-FeatureAssessment -Ctx $ctx -Id 'good.million_events' -Level 'good' -Category 'performance' -Requirement '1,000,000 events mode or benchmark exists' -Implementation $millionImplementation -Conformance $millionConformance -Evidence @('outputs/million_runtime_evidence.json', 'outputs/million_scan.json', 'outputs/million_index.json', 'outputs/million_compare.md') -Details $millionDetails
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.markdown_compare' -Level 'good' -Category 'report' -Requirement 'compare Markdown report' -CommandName 'cli_compare' -RequiredArtifacts @($compare)
Add-FeatureAssessment -Ctx $ctx -Id 'good.index_build_duration' -Level 'good' -Category 'performance' -Requirement 'Index build duration is measured' -Implementation $indexBuildImplementation -Conformance $indexBuildConformance -Evidence @('outputs/index_build_duration_evidence.json', 'outputs/million_runtime_evidence.json', 'outputs/million_index.json', 'outputs/million_compare.md') -Details $indexBuildDetails
Add-CommandFeatureAssessment -Ctx $ctx -Id 'good.benchmarks' -Level 'good' -Category 'performance' -Requirement 'make bench runs query benchmarks' -CommandName 'make_bench'

Add-FeatureAssessment -Ctx $ctx -Id 'excellent.string_query' -Level 'excellent' -Category 'query' -Requirement 'String query language' -Implementation $stringQueryImplementation -Conformance $stringQueryConformance -Evidence @('outputs/string_query_evidence.json', 'outputs/string_query_scan.json', 'outputs/string_query_index.json') -Details $stringQueryDetails
Add-FeatureAssessment -Ctx $ctx -Id 'excellent.intersection_order' -Level 'excellent' -Category 'algorithm' -Requirement 'Intersection order optimized by list length' -Implementation $intersectionImplementation -Conformance $intersectionConformance -Evidence @('outputs/intersection_order_evidence.json', 'outputs/intersection_order_index.json', 'outputs/intersection_order_events.jsonl') -Details $intersectionDetails
Add-FeatureAssessment -Ctx $ctx -Id 'excellent.memory_estimate' -Level 'excellent' -Category 'performance' -Requirement 'Index memory estimate' -Implementation $memoryImplementation -Conformance $memoryConformance -Evidence @('outputs/memory_runtime_evidence.json', 'outputs/million_runtime_evidence.json', 'outputs/million_index.json') -Details $memoryDetails
Add-FeatureAssessment -Ctx $ctx -Id 'excellent.index_tradeoffs' -Level 'excellent' -Category 'documentation' -Requirement 'Index versus scan tradeoffs are documented' -Implementation $docsImplementation -Conformance $docsConformance -Evidence @('outputs/index_tradeoffs_evidence.json', 'repo_snapshot/README.md', 'repo_snapshot/docs/reshenie.md') -Details $docsDetails

Complete-Check -Ctx $ctx -Extra @{
    expected_cli = 'reverse-index-query generate/run/compare'
    expected_outputs = @('scan.json', 'index.json', 'compare.md', 'scan_index_comparison.json')
}



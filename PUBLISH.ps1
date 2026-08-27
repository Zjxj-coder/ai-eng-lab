# =============================================================================
#  把 ai-eng-lab 发到 GitHub 并开启 Pages
#
#  用法（在本目录下）：
#      pwsh -File .\PUBLISH.ps1 -User <你的GitHub用户名>
#
#  ★为什么这一步必须你自己跑★
#  推送需要你的 GitHub 凭据。登录属于你的账号操作，助手不碰凭据，
#  所以这里只把流程压到一条命令，认证那一步由 gh 打开浏览器让你自己完成。
# =============================================================================
param(
    [string]$User = 'Zjxj-coder',
    [string]$Repo = 'ai-eng-lab',
    [switch]$Private
)
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

function Have($n) { [bool](Get-Command $n -ErrorAction SilentlyContinue) }

# ---- 1. 确保 gh 存在 -------------------------------------------------------
if (-not (Have gh)) {
    Write-Host '[1/5] 未检测到 GitHub CLI，正在安装…' -ForegroundColor Yellow
    if (-not (Have winget)) { throw '没有 winget，请手动安装 GitHub CLI: https://cli.github.com/' }
    winget install --id GitHub.cli --silent --accept-package-agreements --accept-source-agreements
    $env:Path = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' +
                [Environment]::GetEnvironmentVariable('Path', 'User')
    if (-not (Have gh)) { throw 'gh 安装后仍不在 PATH，请开一个新终端再跑一次本脚本。' }
}
else { Write-Host '[1/5] GitHub CLI 已就绪' -ForegroundColor Green }

# ---- 2. 认证（浏览器里由你完成）-------------------------------------------
$authed = $false
try { gh auth status 2>&1 | Out-Null; $authed = ($LASTEXITCODE -eq 0) } catch {}
if (-not $authed) {
    Write-Host '[2/5] 需要登录 GitHub —— 浏览器会打开，请你自己完成授权' -ForegroundColor Yellow
    gh auth login --hostname github.com --git-protocol https --web
    gh auth status 2>&1 | Out-Null
    if ($LASTEXITCODE -ne 0) { throw '登录未完成，脚本中止。' }
}
else { Write-Host '[2/5] 已登录 GitHub' -ForegroundColor Green }

# ---- 3. 把 README 与站点里的用户名占位符换成真实用户名 --------------------
Write-Host "[3/5] 写入用户名 $User …" -ForegroundColor Cyan
$pagesUrl = "https://$User.github.io/$Repo/"
foreach ($f in @('README.md')) {
    if (Test-Path $f) {
        $t = [System.IO.File]::ReadAllText($f)
        $t = $t -replace 'https://guojunhao-dev\.github\.io/ai-eng-lab/', $pagesUrl
        $t = $t -replace '（部署后请把上面这行的用户名替换成你自己的 GitHub 用户名）', ''
        [System.IO.File]::WriteAllText($f, $t, (New-Object System.Text.UTF8Encoding($false)))
    }
}

# ---- 4. 建仓 + 推送 --------------------------------------------------------
Write-Host '[4/5] 创建仓库并推送…' -ForegroundColor Cyan
git add -A
git -c user.name="$User" commit -q -m "docs: point site url at $pagesUrl" 2>&1 | Out-Null
$vis = if ($Private) { '--private' } else { '--public' }
gh repo create "$User/$Repo" $vis --source=. --remote=origin --push

# ---- 5. 开启 Pages（从 main 分支的 /docs 目录）-----------------------------
Write-Host '[5/5] 开启 GitHub Pages（main 分支 /docs）…' -ForegroundColor Cyan
gh api -X POST "repos/$User/$Repo/pages" -f "source[branch]=main" -f "source[path]=/docs" 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
    gh api -X PUT "repos/$User/$Repo/pages" -f "source[branch]=main" -f "source[path]=/docs" 2>&1 | Out-Null
}

Write-Host ''
Write-Host '完成。' -ForegroundColor Green
Write-Host "仓库:  https://github.com/$User/$Repo"
Write-Host "作品页: $pagesUrl"
Write-Host ''
Write-Host '★首次部署要等 1-3 分钟才生效。★' -ForegroundColor Yellow
Write-Host '★填进网易官网之前，务必用无痕窗口打开上面这个作品页确认能正常显示。★' -ForegroundColor Yellow

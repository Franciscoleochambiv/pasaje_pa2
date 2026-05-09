param(
    [string]$OutDir = "C:\sistemas\pasaje\evidencias_anexos\capturas_escritorio",
    [int]$WaitSeconds = 6
)

$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName System.Windows.Forms

$targets = @(
    @{ Name = "01_github_repositorio"; Title = "Repositorio GitHub"; Url = "https://github.com/Franciscoleochambiv/pasaje_pa2" },
    @{ Name = "02_github_commits"; Title = "Historial de commits GitHub"; Url = "https://github.com/Franciscoleochambiv/pasaje_pa2/commits/main" },
    @{ Name = "03_despliegue_url_publica"; Title = "Sistema desplegado"; Url = "https://pasaje.facturame.online" },
    @{ Name = "04_login_publico"; Title = "Login publico"; Url = "https://pasaje.facturame.online/login" },
    @{ Name = "05_health_ready_db"; Title = "Health ready con base de datos"; Url = "https://pasaje.facturame.online/health/ready" },
    @{ Name = "06_viajes_publico"; Title = "Modulo viajes publico"; Url = "https://pasaje.facturame.online/viajes" },
    @{ Name = "07_consulta_reserva"; Title = "Consulta de reserva"; Url = "https://pasaje.facturame.online/consulta" },
    @{ Name = "08_admin_sin_token"; Title = "Admin sin token redirige a login"; Url = "https://pasaje.facturame.online/admin" },
    @{ Name = "09_zap_reporte_local"; Title = "Reporte OWASP ZAP local"; Url = "file:///C:/sistemas/pasaje/2026-05-06-ZAP-Report-.html" }
)

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

function Get-BrowserPath {
    $candidates = @(
        "$env:LOCALAPPDATA\ms-playwright\chromium-1217\chrome-win\chrome.exe",
        "$env:ProgramFiles\Google\Chrome\Application\chrome.exe",
        "${env:ProgramFiles(x86)}\Google\Chrome\Application\chrome.exe",
        "$env:ProgramFiles\Microsoft\Edge\Application\msedge.exe",
        "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe"
    )
    foreach ($candidate in $candidates) {
        if ($candidate -and (Test-Path -LiteralPath $candidate)) {
            return $candidate
        }
    }
    throw "No se encontro Chrome, Edge ni Chromium de Playwright."
}

function Capture-Screen([string]$path) {
    $bounds = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds
    $bmp = New-Object System.Drawing.Bitmap($bounds.Width, $bounds.Height)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.CopyFromScreen($bounds.Location, [System.Drawing.Point]::Empty, $bounds.Size)
    $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
    $g.Dispose()
    $bmp.Dispose()
}

function Add-ClockMagnifier([string]$src, [string]$dest, [string]$title) {
    $bmp = [System.Drawing.Bitmap]::FromFile($src)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
    $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::NearestNeighbor

    $w = $bmp.Width
    $h = $bmp.Height

    # Region where Windows normally shows clock/date in the taskbar.
    $cropW = [Math]::Min(430, $w)
    $cropH = [Math]::Min(120, $h)
    $srcRect = New-Object System.Drawing.Rectangle(($w - $cropW), ($h - $cropH), $cropW, $cropH)

    $scale = 2.35
    $dstW = [int]($cropW * $scale)
    $dstH = [int]($cropH * $scale)
    if ($dstW -gt ($w - 80)) {
        $scale = ($w - 80) / $cropW
        $dstW = [int]($cropW * $scale)
        $dstH = [int]($cropH * $scale)
    }
    $dstX = [int](($w - $dstW) / 2)
    $dstY = 34
    $dstRect = New-Object System.Drawing.Rectangle($dstX, $dstY, $dstW, $dstH)

    $shadow = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(130, 0, 0, 0))
    $g.FillRectangle($shadow, $dstX + 10, $dstY + 10, $dstW, $dstH)

    $bg = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::White)
    $g.FillRectangle($bg, $dstRect)
    $g.DrawImage($bmp, $dstRect, $srcRect, [System.Drawing.GraphicsUnit]::Pixel)

    $pen = New-Object System.Drawing.Pen([System.Drawing.Color]::FromArgb(220, 0, 0, 0), 5)
    $g.DrawRectangle($pen, $dstRect)

    $font = New-Object System.Drawing.Font("Arial", 18, [System.Drawing.FontStyle]::Bold)
    $label = "Lupa de fecha y hora real de Windows - $title"
    $labelBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::Black)
    $labelBg = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(235, 255, 255, 255))
    $g.FillRectangle($labelBg, $dstX, $dstY + $dstH + 6, $dstW, 38)
    $g.DrawString($label, $font, $labelBrush, $dstX + 12, $dstY + $dstH + 10)

    # Mark original taskbar clock area so it is clear the magnifier comes from that corner.
    $markPen = New-Object System.Drawing.Pen([System.Drawing.Color]::Red, 4)
    $g.DrawRectangle($markPen, $srcRect)

    $bmp.Save($dest, [System.Drawing.Imaging.ImageFormat]::Png)
    $g.Dispose()
    $bmp.Dispose()
}

$browser = Get-BrowserPath
$userData = Join-Path $OutDir "browser-profile"
New-Item -ItemType Directory -Force -Path $userData | Out-Null

$manifest = @()
$i = 0
foreach ($target in $targets) {
    $i++
    $raw = Join-Path $OutDir ($target.Name + "_original_windows.png")
    $zoom = Join-Path $OutDir ($target.Name + "_lupa_fecha_hora.png")

    $args = @(
        "--new-window",
        "--start-maximized",
        "--user-data-dir=$userData",
        "--disable-first-run-ui",
        "--no-first-run",
        $target.Url
    )

    $proc = Start-Process -FilePath $browser -ArgumentList $args -PassThru
    Start-Sleep -Seconds $WaitSeconds
    Capture-Screen $raw
    Add-ClockMagnifier $raw $zoom $target.Title

    try {
        Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    } catch {
        # Best effort cleanup. The captured evidence is already saved.
    }

    $manifest += [pscustomobject]@{
        name = $target.Name
        title = $target.Title
        url = $target.Url
        original = $raw
        magnifier = $zoom
        captured_at = (Get-Date).ToString("yyyy-MM-dd HH:mm:ss")
    }
}

$manifestPath = Join-Path $OutDir "capturas_escritorio_manifest.json"
$manifest | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $manifestPath -Encoding UTF8

Write-Output "Capturas de escritorio guardadas en: $OutDir"
Write-Output "Manifest: $manifestPath"

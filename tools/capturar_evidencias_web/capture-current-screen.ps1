param(
    [Parameter(Mandatory=$true)][string]$Out,
    [Parameter(Mandatory=$true)][string]$Title
)

$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName System.Windows.Forms

$dir = Split-Path -Parent $Out
New-Item -ItemType Directory -Force -Path $dir | Out-Null

$raw = $Out -replace '\.png$', '_original_windows.png'

$bounds = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds
$bmp = New-Object System.Drawing.Bitmap($bounds.Width, $bounds.Height)
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.CopyFromScreen($bounds.Location, [System.Drawing.Point]::Empty, $bounds.Size)
$bmp.Save($raw, [System.Drawing.Imaging.ImageFormat]::Png)
$g.Dispose()
$bmp.Dispose()

$bmp = [System.Drawing.Bitmap]::FromFile($raw)
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
$g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::NearestNeighbor

$w = $bmp.Width
$h = $bmp.Height
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
$g.FillRectangle([System.Drawing.Brushes]::White, $dstRect)
$g.DrawImage($bmp, $dstRect, $srcRect, [System.Drawing.GraphicsUnit]::Pixel)
$g.DrawRectangle((New-Object System.Drawing.Pen([System.Drawing.Color]::FromArgb(220, 0, 0, 0), 5)), $dstRect)

$font = New-Object System.Drawing.Font("Arial", 18, [System.Drawing.FontStyle]::Bold)
$label = "Lupa de fecha y hora real de Windows - $Title"
$g.FillRectangle((New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(235, 255, 255, 255))), $dstX, $dstY + $dstH + 6, $dstW, 38)
$g.DrawString($label, $font, [System.Drawing.Brushes]::Black, $dstX + 12, $dstY + $dstH + 10)
$g.DrawRectangle((New-Object System.Drawing.Pen([System.Drawing.Color]::Red, 4)), $srcRect)

$bmp.Save($Out, [System.Drawing.Imaging.ImageFormat]::Png)
$g.Dispose()
$bmp.Dispose()

Write-Output $Out

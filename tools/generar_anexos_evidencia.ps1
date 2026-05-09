param(
    [string]$Root = (Resolve-Path ".").Path,
    [string]$OutDir = "evidencias_anexos"
)

$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem

$outRoot = Join-Path $Root $OutDir
$imgDir = Join-Path $outRoot "imagenes"
New-Item -ItemType Directory -Force -Path $imgDir | Out-Null

function XmlEscape([string]$s) {
    if ($null -eq $s) { return "" }
    return [System.Security.SecurityElement]::Escape($s)
}

function Get-RelPath([string]$path) {
    return $path.Replace($Root, "").TrimStart("\", "/")
}

function Mask-Secrets([string[]]$lines) {
    $masked = @()
    foreach ($line in $lines) {
        $l = $line
        if ($l -match "(?i)(PASSWORD|SECRET|TOKEN|KEY|CREDENTIAL|SMTP_PASS|CULQI).*=.+") {
            $l = ($l -replace "=(.*)$", "=********")
        }
        $masked += $l
    }
    return $masked
}

function Read-LinesSafe([string]$rel, [int]$skip = 0, [int]$take = 42) {
    $path = Join-Path $Root $rel
    if (-not (Test-Path $path)) { return @("Archivo no encontrado: $rel") }
    $lines = Get-Content -LiteralPath $path -Encoding UTF8
    return @(($lines | Select-Object -Skip $skip -First $take))
}

function Read-MatchWindow([string]$rel, [string]$pattern, [int]$before = 8, [int]$after = 28) {
    $path = Join-Path $Root $rel
    if (-not (Test-Path $path)) { return @("Archivo no encontrado: $rel") }
    $lines = @(Get-Content -LiteralPath $path -Encoding UTF8)
    $idx = -1
    for ($i = 0; $i -lt $lines.Count; $i++) {
        if ($lines[$i] -match $pattern) { $idx = $i; break }
    }
    if ($idx -lt 0) { return @("Patrón no encontrado: $pattern", "Archivo: $rel") + ($lines | Select-Object -First 35) }
    $start = [Math]::Max(0, $idx - $before)
    $end = [Math]::Min($lines.Count - 1, $idx + $after)
    return @($lines[$start..$end])
}

function New-TextImage {
    param(
        [string]$FileName,
        [string]$Title,
        [string]$Source,
        [string[]]$Lines
    )
    $path = Join-Path $imgDir $FileName
    $width = 1800
    $height = 1050
    $bmp = New-Object System.Drawing.Bitmap($width, $height)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
    $g.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::ClearTypeGridFit

    $bg = [System.Drawing.Color]::FromArgb(22, 27, 34)
    $panel = [System.Drawing.Color]::FromArgb(13, 17, 23)
    $header = [System.Drawing.Color]::FromArgb(35, 134, 54)
    $text = [System.Drawing.Color]::FromArgb(230, 237, 243)
    $muted = [System.Drawing.Color]::FromArgb(139, 148, 158)
    $lineNo = [System.Drawing.Color]::FromArgb(110, 118, 129)
    $accent = [System.Drawing.Color]::FromArgb(88, 166, 255)

    $g.Clear($bg)
    $g.FillRectangle((New-Object System.Drawing.SolidBrush($panel)), 38, 38, $width - 76, $height - 76)
    $g.FillRectangle((New-Object System.Drawing.SolidBrush($header)), 38, 38, $width - 76, 84)

    $titleFont = New-Object System.Drawing.Font("Segoe UI", 28, [System.Drawing.FontStyle]::Bold)
    $metaFont = New-Object System.Drawing.Font("Segoe UI", 16, [System.Drawing.FontStyle]::Regular)
    $codeFont = New-Object System.Drawing.Font("Consolas", 18, [System.Drawing.FontStyle]::Regular)

    $g.DrawString($Title, $titleFont, (New-Object System.Drawing.SolidBrush([System.Drawing.Color]::White)), 64, 54)
    $meta = "$Source  |  generado: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
    $g.DrawString($meta, $metaFont, (New-Object System.Drawing.SolidBrush($muted)), 64, 132)

    $y = 178
    $lineHeight = 27
    $maxLines = [Math]::Floor(($height - 220) / $lineHeight)
    $safeLines = @(Mask-Secrets $Lines)
    if ($safeLines.Count -gt $maxLines) {
        $safeLines = @($safeLines | Select-Object -First ($maxLines - 1)) + @("... contenido recortado para la captura ...")
    }

    for ($i = 0; $i -lt $safeLines.Count; $i++) {
        $num = ($i + 1).ToString().PadLeft(3)
        $g.DrawString($num, $codeFont, (New-Object System.Drawing.SolidBrush($lineNo)), 64, $y)
        $line = $safeLines[$i].Replace("`t", "    ")
        $brush = New-Object System.Drawing.SolidBrush($text)
        if ($line -match "UNIQUE|FOR UPDATE|BEGIN|COMMIT|JWT|bcrypt|WebSocket|postgres|pgxpool|CREATE TABLE|INSERT INTO|UPDATE") {
            $brush = New-Object System.Drawing.SolidBrush($accent)
        }
        $g.DrawString($line, $codeFont, $brush, 128, $y)
        $y += $lineHeight
    }

    $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
    $g.Dispose()
    $bmp.Dispose()
    return $path
}

$images = New-Object System.Collections.Generic.List[object]
function Add-EvidenceImage([string]$name, [string]$title, [string]$source, [string[]]$lines, [string]$caption) {
    $path = New-TextImage -FileName $name -Title $title -Source $source -Lines $lines
    $images.Add([pscustomobject]@{ Path = $path; Title = $title; Caption = $caption }) | Out-Null
}

$tree = @(Get-ChildItem -Path $Root -Directory | Where-Object { $_.Name -notin @(".git", "evidencias_anexos") } | ForEach-Object { $_.Name + "\" })
$tree += @(Get-ChildItem -Path $Root -File | Select-Object -ExpandProperty Name)
Add-EvidenceImage "01_repositorio_estructura.png" "Repositorio y estructura general" "C:\sistemas\pasaje" $tree "Figura 1. Estructura general del repositorio del proyecto Pasaje."

$gitLog = @(git -C $Root log --oneline -n 12 2>&1)
Add-EvidenceImage "02_historial_commits.png" "Historial de commits" "git log --oneline -n 12" $gitLog "Figura 2. Historial reciente de commits del repositorio."

Add-EvidenceImage "03_env_example.png" "Variables de entorno sin credenciales reales" "backend\config\.env.example" (Read-LinesSafe "backend\config\.env.example" 0 48) "Figura 3. Variables de entorno de ejemplo para conexion a PostgreSQL y configuracion del backend."
Add-EvidenceImage "04_config_go.png" "Carga de configuracion" "backend\internal\config\config.go" (Read-MatchWindow "backend\internal\config\config.go" "DBHost|DatabaseURL|Load|os.Getenv" 4 42) "Figura 4. Carga de variables de entorno para configurar la conexion del sistema."
Add-EvidenceImage "05_db_go.png" "Pool de conexion PostgreSQL" "backend\pkg\db\db.go" (Read-MatchWindow "backend\pkg\db\db.go" "pgxpool|Connect|Ping|DATABASE_URL" 4 42) "Figura 5. Creacion del pool de conexion a PostgreSQL mediante pgxpool."
Add-EvidenceImage "06_main_go.png" "Inicializacion del backend" "backend\cmd\server\main.go" (Read-MatchWindow "backend\cmd\server\main.go" "db|NewPool|ListenAndServe|router" 6 48) "Figura 6. Inicializacion del backend, conexion a base de datos y rutas HTTP."

Add-EvidenceImage "07_auth_handler.png" "Modulo de autenticacion" "backend\internal\handler\auth.go" (Read-MatchWindow "backend\internal\handler\auth.go" "Login|JWT|token|bcrypt|password" 5 44) "Figura 7. Handler de autenticacion para login y emision de token."
Add-EvidenceImage "08_auth_service.png" "Servicio de autenticacion" "backend\internal\service\auth.go" (Read-MatchWindow "backend\internal\service\auth.go" "bcrypt|Generate|jwt|Compare" 5 44) "Figura 8. Servicio de autenticacion con validacion de contrasena y generacion de token."
Add-EvidenceImage "09_users_roles.png" "Usuarios y roles" "backend\internal\handler\user.go" (Read-MatchWindow "backend\internal\handler\user.go" "role|admin|operator|List|Create" 5 44) "Figura 9. Gestión de usuarios y roles de administración u operación."
Add-EvidenceImage "10_routes_stops.png" "Rutas y paradas" "backend\internal\handler\routes.go" (Read-MatchWindow "backend\internal\handler\routes.go" "Route|Stop|Create|List" 5 44) "Figura 10. Módulo de rutas y paradas del sistema."
Add-EvidenceImage "11_vehicles_seats.png" "Vehiculos y asientos" "backend\internal\repository\vehicle.go" (Read-MatchWindow "backend\internal\repository\vehicle.go" "vehicle_seats|capacity|INSERT|Create" 5 48) "Figura 11. Registro de vehiculos y relacion con asientos."
Add-EvidenceImage "12_trips.png" "Viajes o salidas" "backend\internal\repository\trip.go" (Read-MatchWindow "backend\internal\repository\trip.go" "trip_instances|departure_at|vehicle_id|INSERT" 5 48) "Figura 12. Creacion de salidas programadas asociadas a ruta, vehiculo y fecha."
Add-EvidenceImage "13_reservations_tx.png" "Reservas con bloqueo transaccional" "backend\internal\repository\reservation.go" (Read-MatchWindow "backend\internal\repository\reservation.go" "FOR UPDATE" 12 48) "Figura 13. Reserva de asientos dentro de una transaccion con bloqueo de filas."
Add-EvidenceImage "14_sales_tickets.png" "Confirmacion de venta y tickets" "backend\internal\repository\reservation.go" (Read-MatchWindow "backend\internal\repository\reservation.go" "UPDATE trip_seat_inventory[\s\S]*sold|INSERT INTO tickets|INSERT INTO payments" 12 58) "Figura 14. Confirmacion de venta: asientos vendidos, tickets y pagos en transaccion."
Add-EvidenceImage "15_websocket.png" "WebSocket de asientos" "backend\internal\handler\ws.go" (Read-MatchWindow "backend\internal\handler\ws.go" "WebSocket|Upgrade|Subscribe|Notify|seats" 5 48) "Figura 15. Conexion WebSocket para reflejar cambios de asientos en tiempo real."

Add-EvidenceImage "16_migration_unique.png" "Migracion critica trip_seat_inventory" "backend\migrations\000002_trip_seat_inventory.up.sql" (Read-LinesSafe "backend\migrations\000002_trip_seat_inventory.up.sql" 0 60) "Figura 16. Restriccion UNIQUE(trip_instance_id, vehicle_seat_id) que evita duplicidad de asientos por viaje."
Add-EvidenceImage "17_migration_reservations.png" "Migracion reservas, tickets y pagos" "backend\migrations\000003_reservations_tickets.up.sql" (Read-LinesSafe "backend\migrations\000003_reservations_tickets.up.sql" 0 70) "Figura 17. Tablas de reservas, items de reserva, tickets y pagos."
Add-EvidenceImage "18_migration_audit.png" "Migracion de auditoria" "backend\migrations\000004_audit_logs.up.sql" (Read-LinesSafe "backend\migrations\000004_audit_logs.up.sql" 0 45) "Figura 18. Tabla de auditoria para cambios criticos."

$zapLines = @()
if (Test-Path (Join-Path $Root "2026-05-06-ZAP-Report-.html")) {
    $zapRaw = Get-Content -LiteralPath (Join-Path $Root "2026-05-06-ZAP-Report-.html") -TotalCount 260
    $zapLines = $zapRaw -replace "<[^>]+>", " " -replace "\s+", " "
} else {
    $zapLines = @("No se encontró el reporte HTML de OWASP ZAP.")
}
Add-EvidenceImage "19_zap_report.png" "Reporte OWASP ZAP" "2026-05-06-ZAP-Report-.html" $zapLines "Figura 19. Evidencia del reporte OWASP ZAP generado para el sistema."

$oldPreference = $ErrorActionPreference
$ErrorActionPreference = "Continue"
Push-Location (Join-Path $Root "backend")
$goResult = @(go test ./... 2>&1)
Pop-Location
$ErrorActionPreference = $oldPreference
Add-EvidenceImage "20_go_test_result.png" "Resultado de pruebas backend" "go test ./..." $goResult "Figura 20. Ejecucion de pruebas backend en el entorno local de evidencias."

$ErrorActionPreference = "Continue"
Push-Location (Join-Path $Root "frontend")
$npmResult = @(npm.cmd run 2>&1)
Pop-Location
$ErrorActionPreference = $oldPreference
Add-EvidenceImage "21_npm_scripts.png" "Scripts disponibles frontend" "npm run" $npmResult "Figura 21. Scripts disponibles del frontend Vue 3."

$desktopCaptureDir = Join-Path $outRoot "capturas_escritorio"
if (Test-Path $desktopCaptureDir) {
    $desktopCaptures = Get-ChildItem -LiteralPath $desktopCaptureDir -Filter "*_lupa_fecha_hora.png" | Sort-Object Name
    foreach ($capture in $desktopCaptures) {
        $baseName = [System.IO.Path]::GetFileNameWithoutExtension($capture.Name)
        $title = "Captura escritorio: " + ($baseName -replace "_lupa_fecha_hora", "" -replace "_", " ")
        $caption = "Evidencia de escritorio con navegador visible, barra de tareas de Windows y lupa sobre la fecha/hora real del sistema: " + $capture.Name
        $images.Add([pscustomobject]@{ Path = $capture.FullName; Title = $title; Caption = $caption }) | Out-Null
    }
}

function New-ParagraphXml([string]$text) {
    return "<w:p><w:r><w:t xml:space=""preserve"">$(XmlEscape $text)</w:t></w:r></w:p>"
}

function New-HeadingXml([string]$text, [int]$level = 1) {
    $style = if ($level -eq 1) { "Heading1" } elseif ($level -eq 2) { "Heading2" } else { "Heading3" }
    return "<w:p><w:pPr><w:pStyle w:val=""$style""/></w:pPr><w:r><w:t>$(XmlEscape $text)</w:t></w:r></w:p>"
}

function ImgPara([string]$rid, [string]$title) {
    $cx = 5486400
    $cy = 3200400
    return @"
<w:p>
  <w:r>
    <w:drawing>
      <wp:inline distT="0" distB="0" distL="0" distR="0" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing">
        <wp:extent cx="$cx" cy="$cy"/>
        <wp:docPr id="1" name="$(XmlEscape $title)"/>
        <a:graphic xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
          <a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">
            <pic:pic xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">
              <pic:nvPicPr><pic:cNvPr id="0" name="$(XmlEscape $title)"/><pic:cNvPicPr/></pic:nvPicPr>
              <pic:blipFill><a:blip r:embed="$rid" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>
              <pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="$cx" cy="$cy"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr>
            </pic:pic>
          </a:graphicData>
        </a:graphic>
      </wp:inline>
    </w:drawing>
  </w:r>
</w:p>
"@
}

$body = ""
$body += New-HeadingXml "Documento complementario de anexos - Sistema Pasaje" 1
$body += New-ParagraphXml "Este documento fue generado como complemento del informe PA4_CHAMBI_ALAMA_PINO_CUETO_HUARI_GAMERO.docx. El documento original no fue modificado."
$body += New-ParagraphXml "Fecha de generacion: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')."
$body += New-ParagraphXml "Repositorio local utilizado: C:\sistemas\pasaje. Repositorio remoto declarado en el informe: https://github.com/Franciscoleochambiv/pasaje_pa2."

$body += New-HeadingXml "Resumen de evidencias agregadas" 1
$body += New-ParagraphXml "Se agregan evidencias de estructura del repositorio, conexion PostgreSQL, implementacion por modulos, migraciones SQL, control transaccional de asientos, WebSocket y reporte OWASP ZAP."
$body += New-ParagraphXml "Las capturas generadas desde archivos locales no muestran contrasenas reales: las variables sensibles se enmascaran con asteriscos."
$body += New-ParagraphXml "No se insertaron capturas del navegador publico, panel de hosting ni pruebas funcionales manuales porque requieren sesion, navegador o acceso al servidor. El documento deja esos espacios indicados para completarlos manualmente si el docente exige imagen real de navegador."

$body += New-HeadingXml "Anexo 1. Repositorio del proyecto y estructura de carpetas" 1
$body += New-ParagraphXml "Evidencia del repositorio local y del historial reciente de commits. Para entrega final, puede complementarse con una captura del navegador mostrando la URL publica de GitHub."

$body += New-HeadingXml "Anexo 2. Codigo de conexion a la base de datos" 1
$body += New-ParagraphXml "La conexion se gestiona con variables de entorno, configuracion centralizada y pool de conexiones PostgreSQL. No se incluyen credenciales reales."

$body += New-HeadingXml "Anexo 3. Codigo de implementacion por modulo" 1
$body += New-ParagraphXml "Se muestran fragmentos de autenticacion, usuarios y roles, rutas, vehiculos, salidas, reservas, ventas/tickets/pagos y WebSocket."
$body += New-ParagraphXml "La logica critica de reservas bloquea filas con SELECT ... FOR UPDATE y confirma la venta dentro de transacciones de PostgreSQL."

$body += New-HeadingXml "Anexo 4. Migraciones SQL" 1
$body += New-ParagraphXml "La restriccion UNIQUE(trip_instance_id, vehicle_seat_id) de trip_seat_inventory es la evidencia principal de que un asiento no puede existir duplicado para una misma salida."

$body += New-HeadingXml "Anexo 5. Evidencias de ejecucion local y pruebas" 1
$body += New-ParagraphXml "Se intento ejecutar go test ./...; el entorno bloqueo escritura en la cache de Go fuera del workspace y descargas desde proxy.golang.org. En frontend no existe script test; se adjuntan los scripts disponibles."

$body += New-HeadingXml "Anexo 6. Evidencias de despliegue web" 1
$body += New-ParagraphXml "El informe declara el dominio https://pasaje.facturame.online. Para cierre final deben agregarse capturas manuales del navegador, certificado HTTPS, login publico, dashboard publico y panel del servidor si estan disponibles."

$body += New-HeadingXml "Anexo 7. Capturas de pruebas funcionales realizadas" 1
$body += New-ParagraphXml "Pendiente manual: login correcto e incorrecto, CRUD de rutas/paradas/vehiculos, generacion de salida, punto de venta, mapa de asientos, reserva, venta, ticket, consulta, WebSocket en dos ventanas, 401 sin token y doble venta bloqueada."

$body += New-HeadingXml "Anexo 8. Evidencias del analisis OWASP ZAP" 1
$body += New-ParagraphXml "Se incluye evidencia del reporte HTML existente de OWASP ZAP. Puede complementarse con capturas directas de la aplicacion ZAP si el docente exige la interfaz abierta."

$rels = ""
$idx = 1
foreach ($img in $images) {
    $rid = "rId$idx"
    $fileName = Split-Path $img.Path -Leaf
    $rels += "<Relationship Id=""$rid"" Type=""http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"" Target=""media/$fileName""/>"
    $body += New-HeadingXml $img.Title 2
    $body += ImgPara $rid $img.Title
    $body += New-ParagraphXml $img.Caption
    $idx++
}

$body += "<w:sectPr><w:pgSz w:w=""12240"" w:h=""15840""/><w:pgMar w:top=""720"" w:right=""720"" w:bottom=""720"" w:left=""720"" w:header=""720"" w:footer=""720"" w:gutter=""0""/></w:sectPr>"

$docXml = @"
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
            xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <w:body>
    $body
  </w:body>
</w:document>
"@

$contentTypes = @"
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="png" ContentType="image/png"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>
"@

$rootRels = @"
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>
"@

$docRels = @"
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  $rels
</Relationships>
"@

$styles = @"
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/></w:style>
  <w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:before="360" w:after="160"/></w:pPr><w:rPr><w:b/><w:sz w:val="32"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:before="240" w:after="120"/></w:pPr><w:rPr><w:b/><w:sz w:val="26"/></w:rPr></w:style>
</w:styles>
"@

$tmp = Join-Path $outRoot "docx_tmp"
if (Test-Path $tmp) { Remove-Item -LiteralPath $tmp -Recurse -Force }
New-Item -ItemType Directory -Force -Path (Join-Path $tmp "_rels") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $tmp "word\_rels") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $tmp "word\media") | Out-Null

Set-Content -LiteralPath (Join-Path $tmp "[Content_Types].xml") -Value $contentTypes -Encoding UTF8
Set-Content -LiteralPath (Join-Path $tmp "_rels\.rels") -Value $rootRels -Encoding UTF8
Set-Content -LiteralPath (Join-Path $tmp "word\document.xml") -Value $docXml -Encoding UTF8
Set-Content -LiteralPath (Join-Path $tmp "word\_rels\document.xml.rels") -Value $docRels -Encoding UTF8
Set-Content -LiteralPath (Join-Path $tmp "word\styles.xml") -Value $styles -Encoding UTF8

foreach ($img in $images) {
    Copy-Item -LiteralPath $img.Path -Destination (Join-Path $tmp ("word\media\" + (Split-Path $img.Path -Leaf))) -Force
}

$docx = Join-Path $outRoot "COMPLEMENTO_ANEXOS_EVIDENCIAS_PASAJE.docx"
if (Test-Path $docx) {
    try {
        Remove-Item -LiteralPath $docx -Force
    } catch {
        $stamp = Get-Date -Format "yyyyMMdd_HHmmss"
        $docx = Join-Path $outRoot "COMPLEMENTO_ANEXOS_EVIDENCIAS_PASAJE_$stamp.docx"
    }
}
$zip = [System.IO.Compression.ZipFile]::Open($docx, [System.IO.Compression.ZipArchiveMode]::Create)
try {
    Get-ChildItem -LiteralPath $tmp -Recurse -File | ForEach-Object {
        $rel = $_.FullName.Substring($tmp.Length).TrimStart("\", "/").Replace("\", "/")
        [System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile($zip, $_.FullName, $rel) | Out-Null
    }
}
finally {
    $zip.Dispose()
}
Remove-Item -LiteralPath $tmp -Recurse -Force

$index = Join-Path $outRoot "INDICE_EVIDENCIAS.txt"
$lines = @(
    "Documento generado: $docx",
    "Imagenes generadas: $imgDir",
    "Documento original no modificado: PA4_CHAMBI_ALAMA_PINO_CUETO_HUARI_GAMERO.docx",
    "",
    "Figuras generadas:"
)
$n = 1
foreach ($img in $images) {
    $lines += ("{0}. {1} - {2}" -f $n, $img.Title, (Get-RelPath $img.Path))
    $n++
}
Set-Content -LiteralPath $index -Value $lines -Encoding UTF8

Write-Output "OK: $docx"
Write-Output "OK: $imgDir"
Write-Output "OK: $index"

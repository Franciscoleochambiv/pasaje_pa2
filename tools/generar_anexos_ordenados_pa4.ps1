param(
    [string]$Root = (Resolve-Path ".").Path,
    [string]$OutDir = "evidencias_anexos"
)

$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem

$outRoot = Join-Path $Root $OutDir
$imgDir = Join-Path $outRoot "imagenes_ordenadas"
New-Item -ItemType Directory -Force -Path $imgDir | Out-Null

function XmlEscape([string]$s) {
    if ($null -eq $s) { return "" }
    return [System.Security.SecurityElement]::Escape($s)
}

function Mask-Secrets([string[]]$lines) {
    $masked = @()
    foreach ($line in $lines) {
        $l = $line
        if ($l -match "(?i)(PASSWORD|SECRET|TOKEN|KEY|CREDENTIAL|SMTP_PASS|CULQI|PRIVATE).*=.+") {
            $l = ($l -replace "=(.*)$", "=********")
        }
        $masked += $l
    }
    return $masked
}

function Read-LinesSafe([string]$rel, [int]$skip = 0, [int]$take = 46) {
    $path = Join-Path $Root $rel
    if (-not (Test-Path $path)) { return @("Archivo no encontrado: $rel") }
    $lines = Get-Content -LiteralPath $path -Encoding UTF8
    return @(($lines | Select-Object -Skip $skip -First $take))
}

function Read-MatchWindow([string]$rel, [string]$pattern, [int]$before = 8, [int]$after = 36) {
    $path = Join-Path $Root $rel
    if (-not (Test-Path $path)) { return @("Archivo no encontrado: $rel") }
    $lines = @(Get-Content -LiteralPath $path -Encoding UTF8)
    $idx = -1
    for ($i = 0; $i -lt $lines.Count; $i++) {
        if ($lines[$i] -match $pattern) { $idx = $i; break }
    }
    if ($idx -lt 0) { return @("Patron no encontrado: $pattern", "Archivo: $rel") + ($lines | Select-Object -First 40) }
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
    $header = [System.Drawing.Color]::FromArgb(31, 111, 235)
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
        if ($line -match "UNIQUE|FOR UPDATE|BEGIN|COMMIT|JWT|bcrypt|WebSocket|postgres|pgxpool|CREATE TABLE|INSERT INTO|UPDATE|Route|Handler|Repository|Service") {
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

function New-PlaceholderImage {
    param([string]$FileName, [string]$Title, [string]$Reason)
    $path = Join-Path $imgDir $FileName
    $width = 1800
    $height = 1050
    $bmp = New-Object System.Drawing.Bitmap($width, $height)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.Clear([System.Drawing.Color]::White)
    $fontTitle = New-Object System.Drawing.Font("Segoe UI", 34, [System.Drawing.FontStyle]::Bold)
    $font = New-Object System.Drawing.Font("Segoe UI", 24, [System.Drawing.FontStyle]::Regular)
    $fontSmall = New-Object System.Drawing.Font("Segoe UI", 20, [System.Drawing.FontStyle]::Bold)
    $g.DrawRectangle((New-Object System.Drawing.Pen([System.Drawing.Color]::Black, 6)), 50, 50, $width - 100, $height - 100)
    $g.DrawString($Title, $fontTitle, [System.Drawing.Brushes]::Black, 90, 110)
    $g.DrawString($Reason, $font, [System.Drawing.Brushes]::Black, 90, 210)
    $g.DrawString("Insertar aqui captura real con fecha y hora del computador.", $fontSmall, [System.Drawing.Brushes]::DarkRed, 90, 350)
    $g.DrawString("Fecha de plantilla: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')", $font, [System.Drawing.Brushes]::Black, 90, 430)
    $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
    $g.Dispose()
    $bmp.Dispose()
    return $path
}

$items = New-Object System.Collections.Generic.List[object]
function Add-ImageItem([string]$section, [string]$title, [string]$path, [string]$caption) {
    $items.Add([pscustomobject]@{ Section=$section; Title=$title; Path=$path; Caption=$caption }) | Out-Null
}
function Add-CodeItem([string]$section, [string]$fileName, [string]$title, [string]$source, [string[]]$lines, [string]$caption) {
    $p = New-TextImage -FileName $fileName -Title $title -Source $source -Lines $lines
    Add-ImageItem $section $title $p $caption
}
function Add-PlaceholderItem([string]$section, [string]$fileName, [string]$title, [string]$reason) {
    $p = New-PlaceholderImage -FileName $fileName -Title $title -Reason $reason
    Add-ImageItem $section $title $p "Pendiente: $title. Debe reemplazarse por captura real con fecha y hora del computador."
}

# Anexo 1. Codigo por modulo
Add-CodeItem "Anexo 1. Código de implementación de módulos" "a1_01_auth_handler.png" "Modulo de autenticacion" "backend\internal\handler\auth.go" (Read-MatchWindow "backend\internal\handler\auth.go" "Login|token|JWT|password" 6 42) "Figura A1-1. Modulo de autenticacion: login y emision de token."
Add-CodeItem "Anexo 1. Código de implementación de módulos" "a1_02_usuarios.png" "Modulo de usuarios" "backend\internal\handler\user.go" (Read-MatchWindow "backend\internal\handler\user.go" "ListUsers|CreateUser|role|UpdateUser" 6 42) "Figura A1-2. Modulo de usuarios: listado, registro y roles."
Add-CodeItem "Anexo 1. Código de implementación de módulos" "a1_03_rutas.png" "Modulo de rutas" "backend\internal\handler\routes.go" (Read-MatchWindow "backend\internal\handler\routes.go" "ListRoutes|Route|routes" 6 42) "Figura A1-3. Modulo de rutas."
Add-CodeItem "Anexo 1. Código de implementación de módulos" "a1_04_paradas.png" "Modulo de paradas" "backend\internal\handler\admin.go" (Read-MatchWindow "backend\internal\handler\admin.go" "ListStops|CreateStop|UpdateStop|DeleteStop" 6 42) "Figura A1-4. Modulo de paradas."
Add-CodeItem "Anexo 1. Código de implementación de módulos" "a1_05_vehiculos.png" "Modulo de vehiculos" "backend\internal\repository\vehicle.go" (Read-MatchWindow "backend\internal\repository\vehicle.go" "vehicle_seats|CreateVehicle|capacity|INSERT" 8 44) "Figura A1-5. Modulo de vehiculos y asientos."
Add-CodeItem "Anexo 1. Código de implementación de módulos" "a1_06_plantillas.png" "Modulo de plantillas" "backend\internal\repository\trip_template.go" (Read-MatchWindow "backend\internal\repository\trip_template.go" "trip_templates|Create|INSERT|List" 8 44) "Figura A1-6. Modulo de plantillas de viaje."
Add-CodeItem "Anexo 1. Código de implementación de módulos" "a1_07_viajes.png" "Modulo de viajes" "backend\internal\repository\trip.go" (Read-MatchWindow "backend\internal\repository\trip.go" "trip_instances|departure_at|vehicle_id|INSERT" 8 44) "Figura A1-7. Modulo de viajes o salidas."
Add-CodeItem "Anexo 1. Código de implementación de módulos" "a1_08_reservas.png" "Modulo de reservas" "backend\internal\repository\reservation.go" (Read-MatchWindow "backend\internal\repository\reservation.go" "FOR UPDATE" 12 44) "Figura A1-8. Modulo de reservas con bloqueo transaccional de asientos."
Add-CodeItem "Anexo 1. Código de implementación de módulos" "a1_09_ventas.png" "Modulo de ventas" "backend\internal\repository\reservation.go" (Read-MatchWindow "backend\internal\repository\reservation.go" "INSERT INTO tickets|INSERT INTO payments|status = 'sold'" 16 52) "Figura A1-9. Modulo de ventas: confirmacion, ticket y pago."
Add-CodeItem "Anexo 1. Código de implementación de módulos" "a1_10_websocket.png" "Modulo de WebSocket" "backend\internal\handler\ws.go" (Read-MatchWindow "backend\internal\handler\ws.go" "HandleTripSeats|WebSocket|Upgrade|Subscribe" 6 44) "Figura A1-10. Modulo WebSocket para cambios de asientos en tiempo real."
Add-CodeItem "Anexo 1. Código de implementación de módulos" "a1_11_consultas.png" "Modulo de consultas" "backend\internal\handler\reservation.go" (Read-MatchWindow "backend\internal\handler\reservation.go" "GetByCode|reservation|code" 6 44) "Figura A1-11. Modulo de consultas de reserva por codigo."

# Anexo 3. Migraciones SQL
$migrations = @(
    @{File="backend\migrations\000001_init_schema.up.sql"; Name="000001_init_schema"},
    @{File="backend\migrations\000002_trip_seat_inventory.up.sql"; Name="000002_trip_seat_inventory"},
    @{File="backend\migrations\000003_reservations_tickets.up.sql"; Name="000003_reservations_tickets"},
    @{File="backend\migrations\000004_audit_logs.up.sql"; Name="000004_audit_logs"},
    @{File="backend\migrations\000005_seed_minimal.up.sql"; Name="000005_seed_minimal"},
    @{File="backend\migrations\000006_vehicle_floors_layout.up.sql"; Name="000006_vehicle_floors_layout"},
    @{File="backend\migrations\000007_auth.up.sql"; Name="000007_auth"}
)
foreach ($m in $migrations) {
    Add-CodeItem "Anexo 3. Migraciones SQL" ("a3_" + $m.Name + ".png") $m.Name $m.File (Read-LinesSafe $m.File 0 52) ("Figura A3. Migracion SQL " + $m.Name + ".")
}

# Anexo 4. Capturas de pruebas realizadas
$desktop = Join-Path $outRoot "capturas_escritorio"
function DesktopCap([string]$file) { return Join-Path $desktop $file }

Add-PlaceholderItem "Anexo 4. Capturas de pruebas realizadas" "a4_01_login_correcto_pendiente.png" "Login correcto" "Requiere credenciales validas de produccion o un usuario de prueba autorizado."
Add-ImageItem "Anexo 4. Capturas de pruebas realizadas" "Login incorrecto" (DesktopCap "04_login_publico_lupa_fecha_hora.png") "Figura A4-2. Login publico con fecha y hora del computador. Complementar con captura de error si se requiere."
if (Test-Path (Join-Path $outRoot "capturas_web\08_login_incorrecto.png")) {
    Add-ImageItem "Anexo 4. Capturas de pruebas realizadas" "Login incorrecto con mensaje de error" (Join-Path $outRoot "capturas_web\08_login_incorrecto.png") "Figura A4-2b. Prueba de login incorrecto mostrando mensaje de credenciales invalidas."
}
Add-PlaceholderItem "Anexo 4. Capturas de pruebas realizadas" "a4_03_registro_ruta_pendiente.png" "Registro de ruta" "Requiere sesion administrativa y crear/listar una ruta real."
Add-PlaceholderItem "Anexo 4. Capturas de pruebas realizadas" "a4_04_registro_vehiculo_pendiente.png" "Registro de vehiculo" "Requiere sesion administrativa y crear/listar un vehiculo real."
Add-PlaceholderItem "Anexo 4. Capturas de pruebas realizadas" "a4_05_generacion_viaje_pendiente.png" "Generacion de viaje" "Requiere sesion administrativa y generar una salida real."
Add-ImageItem "Anexo 4. Capturas de pruebas realizadas" "Mapa de asientos / viajes publico" (DesktopCap "06_viajes_publico_lupa_fecha_hora.png") "Figura A4-6. Modulo publico de viajes con fecha y hora del computador."
Add-PlaceholderItem "Anexo 4. Capturas de pruebas realizadas" "a4_07_reserva_temporal_pendiente.png" "Reserva temporal" "Requiere ejecutar el flujo real de seleccion de asiento y reserva."
Add-PlaceholderItem "Anexo 4. Capturas de pruebas realizadas" "a4_08_venta_confirmada_pendiente.png" "Venta confirmada" "Requiere confirmar pago o venta desde el sistema."
Add-PlaceholderItem "Anexo 4. Capturas de pruebas realizadas" "a4_09_ticket_generado_pendiente.png" "Ticket generado" "Requiere una venta confirmada con comprobante/ticket visible."
Add-ImageItem "Anexo 4. Capturas de pruebas realizadas" "Consulta de reserva" (DesktopCap "07_consulta_reserva_lupa_fecha_hora.png") "Figura A4-10. Pantalla de consulta de reserva con fecha y hora del computador."
Add-PlaceholderItem "Anexo 4. Capturas de pruebas realizadas" "a4_11_websocket_pendiente.png" "Actualizacion por WebSocket" "Requiere dos ventanas abiertas mostrando cambio de estado del asiento."
Add-PlaceholderItem "Anexo 4. Capturas de pruebas realizadas" "a4_12_doble_venta_pendiente.png" "Prueba de doble venta" "Requiere intentar tomar el mismo asiento desde dos flujos y capturar el bloqueo."
Add-ImageItem "Anexo 4. Capturas de pruebas realizadas" "Prueba de acceso sin token" (DesktopCap "08_admin_sin_token_lupa_fecha_hora.png") "Figura A4-13. Acceso a /admin sin token redirigido al login."
Add-ImageItem "Anexo 4. Capturas de pruebas realizadas" "Base de datos validando registro" (DesktopCap "05_health_ready_db_lupa_fecha_hora.png") "Figura A4-14. Health ready evidencia conexion a base de datos: database connected."

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
$body += New-HeadingXml "Anexos" 1
$body += New-ParagraphXml "Documento complementario generado en el orden indicado por el informe. El archivo Word original no fue modificado."
$body += New-ParagraphXml "Fecha de generacion: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')."

$current = ""
$rels = ""
$idx = 1
foreach ($item in $items) {
    if ($item.Section -ne $current) {
        $current = $item.Section
        $body += New-HeadingXml $current 1
        if ($current -like "Anexo 1*") {
            $body += New-ParagraphXml "[INSERTAR CODIGO O CAPTURAS DEL REPOSITORIO POR MODULO]"
        } elseif ($current -like "Anexo 3*") {
            $body += New-ParagraphXml "[INSERTAR MIGRACIONES SQL]"
        } elseif ($current -like "Anexo 4*") {
            $body += New-ParagraphXml "[INSERTAR CAPTURAS CON FECHA Y HORA DEL COMPUTADOR]"
        }
    }
    $rid = "rId$idx"
    $fileName = Split-Path $item.Path -Leaf
    $rels += "<Relationship Id=""$rid"" Type=""http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"" Target=""media/$fileName""/>"
    $body += New-HeadingXml $item.Title 2
    $body += ImgPara $rid $item.Title
    $body += New-ParagraphXml $item.Caption
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

$tmp = Join-Path $outRoot "docx_ordenado_tmp"
if (Test-Path $tmp) { Remove-Item -LiteralPath $tmp -Recurse -Force }
New-Item -ItemType Directory -Force -Path (Join-Path $tmp "_rels") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $tmp "word\_rels") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $tmp "word\media") | Out-Null
Set-Content -LiteralPath (Join-Path $tmp "[Content_Types].xml") -Value $contentTypes -Encoding UTF8
Set-Content -LiteralPath (Join-Path $tmp "_rels\.rels") -Value $rootRels -Encoding UTF8
Set-Content -LiteralPath (Join-Path $tmp "word\document.xml") -Value $docXml -Encoding UTF8
Set-Content -LiteralPath (Join-Path $tmp "word\_rels\document.xml.rels") -Value $docRels -Encoding UTF8
Set-Content -LiteralPath (Join-Path $tmp "word\styles.xml") -Value $styles -Encoding UTF8
foreach ($item in $items) {
    Copy-Item -LiteralPath $item.Path -Destination (Join-Path $tmp ("word\media\" + (Split-Path $item.Path -Leaf))) -Force
}

$docx = Join-Path $outRoot "ANEXOS_ORDENADOS_PA4_PASAJE.docx"
if (Test-Path $docx) {
    try { Remove-Item -LiteralPath $docx -Force }
    catch { $docx = Join-Path $outRoot ("ANEXOS_ORDENADOS_PA4_PASAJE_" + (Get-Date -Format "yyyyMMdd_HHmmss") + ".docx") }
}
$zip = [System.IO.Compression.ZipFile]::Open($docx, [System.IO.Compression.ZipArchiveMode]::Create)
try {
    Get-ChildItem -LiteralPath $tmp -Recurse -File | ForEach-Object {
        $rel = $_.FullName.Substring($tmp.Length).TrimStart("\", "/").Replace("\", "/")
        [System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile($zip, $_.FullName, $rel) | Out-Null
    }
}
finally { $zip.Dispose() }
Remove-Item -LiteralPath $tmp -Recurse -Force

$index = Join-Path $outRoot "INDICE_ANEXOS_ORDENADOS.txt"
$lines = @("Documento generado: $docx", "Imagenes generadas: $imgDir", "Total de figuras: $($items.Count)", "")
$n = 1
foreach ($item in $items) {
    $lines += ("{0}. [{1}] {2}" -f $n, $item.Section, $item.Title)
    $n++
}
Set-Content -LiteralPath $index -Value $lines -Encoding UTF8

Write-Output "OK: $docx"
Write-Output "OK: $index"

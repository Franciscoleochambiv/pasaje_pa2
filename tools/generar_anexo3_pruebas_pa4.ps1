param(
    [string]$Root = (Resolve-Path ".").Path,
    [string]$OutDir = "evidencias_anexos"
)

$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem

$outRoot = Join-Path $Root $OutDir
$capDir = Join-Path $outRoot "capturas_pruebas_pa4"

function XmlEscape([string]$s) {
    if ($null -eq $s) { return "" }
    return [System.Security.SecurityElement]::Escape($s)
}

function New-ParagraphXml([string]$text) {
    return "<w:p><w:r><w:t xml:space=""preserve"">$(XmlEscape $text)</w:t></w:r></w:p>"
}

function New-HeadingXml([string]$text, [int]$level = 1) {
    $style = if ($level -eq 1) { "Heading1" } else { "Heading2" }
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

$items = @(
    @{ Title="Login correcto"; File="01_login_correcto_lupa_fecha_hora.png"; Caption="Figura 1. Login correcto en el panel administrativo del sistema Pasaje con fecha y hora del computador." },
    @{ Title="Login incorrecto"; File="02_login_incorrecto_lupa_fecha_hora.png"; Caption="Figura 2. Login incorrecto mostrando validacion de credenciales invalidas con fecha y hora del computador." },
    @{ Title="Registro de ruta"; File="03_registro_ruta_lupa_fecha_hora.png"; Caption="Figura 3. Registro/listado de ruta creada para evidencia PA4." },
    @{ Title="Registro de vehiculo"; File="04_registro_vehiculo_lupa_fecha_hora.png"; Caption="Figura 4. Registro/listado de vehiculo creado para evidencia PA4." },
    @{ Title="Generacion de viaje"; File="05_generacion_viaje_lupa_fecha_hora.png"; Caption="Figura 5. Generacion/listado de viaje programado asociado a la ruta y vehiculo de evidencia." },
    @{ Title="Mapa de asientos"; File="06_mapa_asientos_lupa_fecha_hora.png"; Caption="Figura 6. Mapa de asientos del viaje de evidencia." },
    @{ Title="Reserva temporal"; File="07_reserva_temporal_lupa_fecha_hora.png"; Caption="Figura 7. Reserva temporal creada con codigo, estado y expiracion." },
    @{ Title="Venta confirmada"; File="08_venta_confirmada_lupa_fecha_hora.png"; Caption="Figura 8. Venta confirmada con estado de reserva confirmado y pago completado." },
    @{ Title="Ticket generado"; File="09_ticket_generado_lupa_fecha_hora.png"; Caption="Figura 9. Ticket generado para la venta confirmada." },
    @{ Title="Consulta de reserva"; File="10_consulta_reserva_lupa_fecha_hora.png"; Caption="Figura 10. Consulta de reserva desde la interfaz publica." },
    @{ Title="Actualizacion por WebSocket"; File="11_websocket_dos_ventanas_lupa_fecha_hora.png"; Caption="Figura 11. Dos ventanas del mapa de asientos para evidenciar actualizacion del estado por WebSocket/refresco del viaje." },
    @{ Title="Prueba de doble venta"; File="12_doble_venta_lupa_fecha_hora.png"; Caption="Figura 12. Prueba de doble venta bloqueada al intentar reservar nuevamente el mismo asiento." },
    @{ Title="Prueba de acceso sin token"; File="13_acceso_sin_token_lupa_fecha_hora.png"; Caption="Figura 13. Acceso a ruta administrativa sin token redirigido a login." },
    @{ Title="Captura de base de datos validando registro"; File="14_base_datos_validando_registro_lupa_fecha_hora.png"; Caption="Figura 14. Evidencia de registros creados y conexion a base de datos validada." }
)

$body = ""
$body += New-HeadingXml "Anexo 3. Capturas de pruebas realizadas" 1
$body += New-ParagraphXml "[INSERTAR CAPTURAS CON FECHA Y HORA DEL COMPUTADOR]"
$body += New-ParagraphXml "Las capturas fueron tomadas desde el escritorio de Windows. Cada imagen conserva la barra de tareas y una lupa sobre la fecha y hora real del sistema."
$body += New-ParagraphXml "Fecha de generacion del anexo: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')."

$rels = ""
$idx = 1
foreach ($item in $items) {
    $path = Join-Path $capDir $item.File
    if (-not (Test-Path -LiteralPath $path)) {
        throw "No existe la captura requerida: $path"
    }
    $rid = "rId$idx"
    $rels += "<Relationship Id=""$rid"" Type=""http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"" Target=""media/$($item.File)""/>"
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
  <w:body>$body</w:body>
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
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">$rels</Relationships>
"@
$styles = @"
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/></w:style>
  <w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:before="360" w:after="160"/></w:pPr><w:rPr><w:b/><w:sz w:val="32"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:before="240" w:after="120"/></w:pPr><w:rPr><w:b/><w:sz w:val="26"/></w:rPr></w:style>
</w:styles>
"@

$tmp = Join-Path $outRoot "docx_anexo3_tmp"
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
    Copy-Item -LiteralPath (Join-Path $capDir $item.File) -Destination (Join-Path $tmp ("word\media\" + $item.File)) -Force
}

$docx = Join-Path $outRoot "ANEXO_3_CAPTURAS_PRUEBAS_PA4.docx"
if (Test-Path $docx) {
    try { Remove-Item -LiteralPath $docx -Force }
    catch { $docx = Join-Path $outRoot ("ANEXO_3_CAPTURAS_PRUEBAS_PA4_" + (Get-Date -Format "yyyyMMdd_HHmmss") + ".docx") }
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

Write-Output "OK: $docx"

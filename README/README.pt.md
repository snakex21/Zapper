**Languages:** [English](../README.md) · [Polski](README.pl.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Español](README.es.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Nederlands](README.nl.md) · [Svenska](README.sv.md) · [Norsk](README.no.md) · [Dansk](README.da.md) · [Suomi](README.fi.md) · [Čeština](README.cs.md) · [Slovenčina](README.sk.md) · [Magyar](README.hu.md) · [Română](README.ro.md) · [Türkçe](README.tr.md) · [Bahasa Indonesia](README.id.md) · [Bahasa Melayu](README.ms.md) · [Tiếng Việt](README.vi.md) · [Русский](README.ru.md) · [Українська](README.uk.md) · [Български](README.bg.md) · [Ελληνικά](README.el.md) · [العربية](README.ar.md) · [עברית](README.he.md) · [हिन्दी](README.hi.md) · [中文](README.zh.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

# Zapper — Go + WebView2

A nova versão da aplicação funciona numa única janela e não requer Python, Node.js nem Wails. Pode ser utilizada como planeador e registo sem uma placa ligada ou para controlar um Arduino Nano através de USB.

## Licença e responsabilidade

O código, o firmware, os esquemas e a documentação estão disponíveis publicamente para utilização não comercial ao abrigo da licença **PolyForm Noncommercial 1.0.0**. Podem ser utilizados, estudados, modificados e distribuídos para os fins permitidos por essa licença, mas o projeto não pode ser utilizado comercialmente sem autorização separada do autor. Consulte o ficheiro `LICENSE` para mais detalhes.

O projeto é disponibilizado sem garantia para experiências independentes e utilização DIY. O utilizador é responsável pela montagem correta, pelas modificações e pela forma como o dispositivo é utilizado. O autor não se responsabiliza por danos no equipamento, outros prejuízos ou consequências de montagem ou utilização incorretas e não garante quaisquer efeitos específicos na saúde.

## Executar a aplicação

Execute `Zapper.exe` a partir da pasta da versão portable. As pessoas persistentes e os respetivos identificadores são guardados em `data/persons.json`, os perfis ativos em `data/profiles.json` e cada execução tem o seu próprio ficheiro em `data/progress/`. As execuções concluídas são movidas para pastas `data/archive/<id>/` que contêm `profile.json` e `progress.json`. As definições da placa são guardadas em `data/device.json`, enquanto as definições da aplicação, incluindo o idioma detetado ou selecionado, ficam em `data/settings.json`. As cópias de segurança permanecem em subpastas locais `backups/`. Tudo fica junto do EXE; nada é escrito em AppData, Documentos ou no Registo do Windows.

Na vista **Perfis** pode adicionar pessoas, gerar texto de contexto para IA pronto a copiar e colar JSON simplificado devolvido por um modelo de IA. As frequências neste formato são indicadas como `frequency_hz`; a aplicação valida o perfil, apresenta uma pré-visualização e só cria um novo `run_id` após confirmação. A execução ativa anterior dessa pessoa é arquivada primeiro.

Durante uma sessão de perfil, o botão **Pausa** guarda a parte restante do passo atual e todos os passos seguintes no progresso local. Ao retomar, é enviada uma sequência abreviada para o firmware inalterado e é novamente necessária uma confirmação física na placa. **Parar** cancela o progresso parcial e deixa a sessão completa disponível para ser executada novamente.

As sessões ignoradas permanecem na fila como atrasadas. As regras do programa definem o número de partes, as pausas dentro de uma série, o intervalo entre sessões completas, o período de recuperação após uma sessão e a compatibilidade com outros programas no mesmo dia. Um perfil sem sessões atrasadas é arquivado automaticamente depois de o plano ser concluído, enquanto **Terminar programa** permite fechá-lo antecipadamente.

## Idioma da aplicação

Ao iniciar, a aplicação lê o idioma do Windows/WebView2 e associa-o a um dos 30 idiomas suportados. Enquanto a definição permanecer no modo **Automático (Windows)**, a deteção é efetuada em cada arranque. Uma escolha manual de idioma no painel esquerdo é guardada em `data/settings.json` e desativa alterações automáticas até o modo automático ser selecionado novamente.

O idioma da aplicação também é o idioma predefinido da variante de firmware. Para sistemas de escrita que um LCD1602/HD44780 normal não consegue apresentar de forma portátil, a aplicação seleciona a variante de firmware correspondente com texto LCD em inglês; a interface de desktop continua a utilizar o idioma escolhido.

## Arduino e USB

O firmware atual encontra-se em `firmware/zapper_v5/zapper_v5.ino` e a respetiva descrição em `firmware/zapper_v5/README.md`. Depois de carregar o firmware:

1. Abra a vista **Dispositivo**.
2. Selecione a porta COM e clique em **Ligar**.
3. Aguarde pelo estado **Pronto**.
4. Envie a sessão de hoje ou inicie um valor único no modo manual.
5. Verifique os terminais na placa e depois prima o botão físico; só então a saída será iniciada.

A porta selecionada é memorizada no ficheiro local `data/device.json`. As sessões de perfil guardam `device_steps` separados e exatos; uma descrição como “30 kHz” continua a ser texto legível por uma pessoa, enquanto a placa recebe `30000000` milihertz e a duração em milissegundos.

### Idiomas do firmware LCD

O firmware 5.1.0 tem 30 variantes linguísticas separadas geradas a partir de uma única base de código. Cada sketch Arduino contém apenas um conjunto de textos LCD. Os idiomas que utilizam o alfabeto latino têm os seus próprios textos curtos guardados em ASCII seguro. Para cirílico e outros sistemas de escrita que um LCD1602/HD44780 típico não consegue apresentar de forma portátil, a variante correspondente utiliza uma interface LCD em inglês. A lista completa encontra-se em `firmware/LANGUAGES.md`.

O comando `go run ./tools/firmware_i18n` gera todos os sketches em `build/generated/firmware/`. O processo normal `build.ps1` faz isto automaticamente e inclui as variantes na versão portable.

### Carregar firmware a partir da aplicação

A secção **Dispositivo → Firmware** mostra a versão detetada, a versão mais recente, o idioma da variante do firmware e o idioma do LCD. O utilizador escolhe o bootloader novo ou antigo do Arduino Nano e clica explicitamente em **Carregar firmware**; a aplicação nunca grava automaticamente a placa ao iniciar.

A compilação e o envio são efetuados por `arduino-cli`. O Zapper procura-o em `tools/arduino-cli/`, junto do EXE, em `PATH` e em localizações típicas do Arduino IDE. Se a ferramenta não estiver disponível, a aplicação indica-o claramente e o botão de gravação permanece desativado. A compilação também requer que o core `arduino:avr` e a biblioteca `LiquidCrystal_I2C` estejam disponíveis para a instalação de `arduino-cli` utilizada.

### Deteção de idioma e seleção de firmware

Ao iniciar, a aplicação lê o idioma do ambiente WebView2/Windows (`navigator.languages`) e associa-o a um dos 30 códigos suportados. Se o idioma do sistema não for suportado, é selecionado o inglês. No modo **Automático (Windows)**, o idioma é verificado em cada arranque; uma seleção manual é guardada em `data/settings.json` até o modo automático ser novamente ativado.

O mesmo código de idioma é a escolha predefinida no ecrã de gravação do firmware. Para idiomas não suportados pelo LCD1602, a aplicação continua a selecionar a variante identificada pelo idioma do utilizador, mas informa que o texto do LCD será em inglês. O firmware nunca é gravado automaticamente ao iniciar a aplicação; a gravação exige um clique explícito do utilizador para evitar substituir acidentalmente outro programa já guardado no Arduino.

## Compilação

É necessário ter Go. A forma mais simples é executar isto na raiz do projeto:

```text
build.bat
```

Em alternativa, no PowerShell:

```powershell
.\build.ps1
```

O script executa testes e análise de código, compila `build/generated/Zapper-dev.exe` e prepara o portable `build/Zapper/Zapper.exe` sem janela de consola.

## Estrutura do projeto

- `app/` — código Go, interface HTML/CSS/JS, guia e base de dados de frequências.
- `firmware/zapper_v5/` — firmware Arduino atual.
- `data/` — perfis ativos, progresso, arquivo, definições do dispositivo e cópias de segurança automáticas.
- `locales/` — traduções versionadas da interface e do guia, usadas no desenvolvimento e copiadas para as versões publicadas.
- `build/Zapper/` — versão portable pronta para copiar para outro computador.
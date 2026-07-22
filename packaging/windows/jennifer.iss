; SPDX-License-Identifier: LGPL-3.0-only
; Copyright (C) 2026 mplx <jennifer@mplx.dev>
;
; Inno Setup script for the Windows installer (best-effort, UNSUPPORTED build).
; Produces jennifer-<version>-setup.exe: a per-user, no-admin installer that
; drops the standard-Go jennifer.exe under %LOCALAPPDATA%\Programs\Jennifer,
; adds it to the user PATH, bundles the Jennifer-coded system modules and points
; JENNIFER_SYSMODDIR at them (so a bare `import "name.j";` resolves - the Unix
; compile-time module dir does not exist on Windows), and optionally associates
; .j files. Compile with:  ISCC.exe /DAppVersion=<tag> packaging\windows\jennifer.iss
; (see packaging/windows/README.md). Requires Inno Setup 6.3+.

#define AppName "Jennifer"
#define AppPublisher "mplx"
#define AppURL "https://jennifer-lang.dev"
#ifndef AppVersion
  #define AppVersion "0.0.0-dev"
#endif
; Repo root, relative to this .iss (packaging/windows/). CI builds jennifer.exe
; at the repo root before invoking ISCC.
#ifndef RepoRoot
  #define RepoRoot "..\.."
#endif

[Setup]
; A stable, installer-unique GUID so upgrades replace in place and the
; uninstaller is found. Generated once; never change it.
AppId={{7C3B1F2E-9A54-4E7D-8B2C-2E6D5A1F0C34}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL={#AppURL}
AppSupportURL={#AppURL}
; Per-user install, no elevation. {autopf} resolves to %LOCALAPPDATA%\Programs
; when PrivilegesRequired=lowest.
DefaultDirName={autopf}\{#AppName}
DefaultGroupName={#AppName}
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
OutputDir={#RepoRoot}\dist
OutputBaseFilename=jennifer-{#AppVersion}-setup
SetupIconFile={#RepoRoot}\docs\favicon.ico
UninstallDisplayIcon={app}\jennifer.ico
WizardStyle=modern
Compression=lzma2
SolidCompression=yes
; The LGPL-3.0 text the release also ships as LICENSE.txt.
LicenseFile={#RepoRoot}\packaging\debian\copyright
; Shown on the "ready to install" page: this is an unsupported build.
DisableWelcomePage=no

[Messages]
WelcomeLabel2=This will install [name/ver] on your computer.%n%nThis is a best-effort, UNSUPPORTED Windows build: the standard jennifer.exe only (no jennifer-tiny), unsigned, and not covered by support. Linux is the only supported platform.

[Tasks]
Name: "addtopath"; Description: "Add Jennifer to your PATH (recommended)"; GroupDescription: "System integration:"
Name: "associatej"; Description: "Associate .j files (adds a ""Run with Jennifer"" right-click action)"; GroupDescription: "System integration:"; Flags: unchecked

[Files]
Source: "{#RepoRoot}\jennifer.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#RepoRoot}\docs\favicon.ico"; DestDir: "{app}"; DestName: "jennifer.ico"; Flags: ignoreversion
Source: "{#RepoRoot}\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#RepoRoot}\JENNIFER.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#RepoRoot}\packaging\UNSUPPORTED.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#RepoRoot}\packaging\debian\copyright"; DestDir: "{app}"; DestName: "LICENSE.txt"; Flags: ignoreversion
; Jennifer-coded system modules (importable via the bundled JENNIFER_SYSMODDIR),
; minus the white-box *_test.j overlays.
Source: "{#RepoRoot}\modules\*.j"; DestDir: "{app}\share\jennifer\modules"; Excludes: "*_test.j"; Flags: ignoreversion

[Icons]
Name: "{group}\Jennifer REPL"; Filename: "{app}\jennifer.exe"; Parameters: "repl"; IconFilename: "{app}\jennifer.ico"; Comment: "Start the interactive Jennifer REPL"
Name: "{group}\Jennifer language reference"; Filename: "{app}\JENNIFER.md"
Name: "{group}\Uninstall Jennifer"; Filename: "{uninstallexe}"

[Registry]
; Point the module search path at the bundled system modules (auto-removed on
; uninstall). Precedence is --sysmoddir > JENNIFER_SYSMODDIR > compile default,
; and the compile default is a POSIX path that never exists on Windows.
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "JENNIFER_SYSMODDIR"; ValueData: "{app}\share\jennifer\modules"; Flags: preservestringtype uninsdeletevalue

; Prepend the install dir to the user PATH, but only if it is not already there
; (idempotent reinstall). Removed surgically on uninstall by CurUninstallStepChanged.
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; ValueData: "{app};{olddata}"; Flags: preservestringtype; Tasks: addtopath; Check: NeedsAddPath(ExpandConstant('{app}'))

; .j association (opt-in). The default double-click action opens the source in
; Notepad - safe; running is an explicit "Run with Jennifer" verb, so a
; double-click never silently executes code.
Root: HKCU; Subkey: "Software\Classes\.j"; ValueType: string; ValueName: ""; ValueData: "Jennifer.Source"; Flags: uninsdeletevalue; Tasks: associatej
Root: HKCU; Subkey: "Software\Classes\Jennifer.Source"; ValueType: string; ValueName: ""; ValueData: "Jennifer source file"; Flags: uninsdeletekey; Tasks: associatej
Root: HKCU; Subkey: "Software\Classes\Jennifer.Source\DefaultIcon"; ValueType: string; ValueName: ""; ValueData: "{app}\jennifer.ico"; Tasks: associatej
Root: HKCU; Subkey: "Software\Classes\Jennifer.Source\shell\open\command"; ValueType: string; ValueName: ""; ValueData: "notepad.exe ""%1"""; Tasks: associatej
Root: HKCU; Subkey: "Software\Classes\Jennifer.Source\shell\run"; ValueType: string; ValueName: ""; ValueData: "Run with Jennifer"; Tasks: associatej
Root: HKCU; Subkey: "Software\Classes\Jennifer.Source\shell\run\command"; ValueType: string; ValueName: ""; ValueData: """{app}\jennifer.exe"" run ""%1"""; Tasks: associatej

[Code]
function SendMessageTimeout(hWnd: HWND; Msg: UINT; wParam: Longint;
  lParam: AnsiString; fuFlags, uTimeout: UINT; var lpdwResult: DWORD): Longint;
  external 'SendMessageTimeoutA@user32.dll stdcall';

{ True when Dir is not already a ;-delimited entry of the user PATH. }
function NeedsAddPath(Param: string): Boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKCU, 'Environment', 'Path', OrigPath) then
  begin
    Result := True;
    exit;
  end;
  Result := Pos(';' + Lowercase(Param) + ';', ';' + Lowercase(OrigPath) + ';') = 0;
end;

{ Tell already-running processes (Explorer, shells) to reload the environment
  block so PATH / JENNIFER_SYSMODDIR take effect without a logout. }
procedure BroadcastEnvChange;
var
  Res: DWORD;
begin
  { HWND_BROADCAST=$FFFF, WM_SETTINGCHANGE=$1A, SMTO_ABORTIFHUNG=$2 - inlined
    because Inno Setup already predefines these Windows constants. }
  SendMessageTimeout($FFFF, $001A, 0, 'Environment', $0002, 5000, Res);
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
    BroadcastEnvChange;
end;

{ Rebuild the user PATH without the install dir (the [Registry] add cannot be
  auto-reverted because it merged into an existing multi-value string). }
procedure RemovePathEntry(const Dir: string);
var
  Path, NewPath, Part: string;
  P: Integer;
begin
  if not RegQueryStringValue(HKCU, 'Environment', 'Path', Path) then
    exit;
  NewPath := '';
  while Length(Path) > 0 do
  begin
    P := Pos(';', Path);
    if P = 0 then
    begin
      Part := Path;
      Path := '';
    end
    else
    begin
      Part := Copy(Path, 1, P - 1);
      Delete(Path, 1, P);
    end;
    if (Part <> '') and (CompareText(Part, Dir) <> 0) then
    begin
      if NewPath <> '' then
        NewPath := NewPath + ';';
      NewPath := NewPath + Part;
    end;
  end;
  RegWriteExpandStringValue(HKCU, 'Environment', 'Path', NewPath);
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usUninstall then
  begin
    RemovePathEntry(ExpandConstant('{app}'));
    BroadcastEnvChange;
  end;
end;

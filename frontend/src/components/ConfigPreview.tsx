import Editor from '@monaco-editor/react'

interface ConfigPreviewProps {
  content: string
  language: 'json' | 'toml' | 'markdown'
}

export function ConfigPreview({ content, language }: ConfigPreviewProps) {
  return (
    <div className="rounded-md border border-border overflow-hidden">
      <div className="bg-muted px-3 py-1.5 border-b border-border">
        <span className="text-xs font-medium text-muted-foreground">
          Preview: {language === 'json' ? 'settings.json' : language === 'toml' ? 'config.toml' : 'GEMINI.md'}
        </span>
      </div>
      <Editor
        height="200px"
        language={language}
        value={content}
        theme="vs-dark"
        options={{
          readOnly: true,
          minimap: { enabled: false },
          lineNumbers: 'off',
          scrollBeyondLastLine: false,
          fontSize: 12,
          wordWrap: 'on',
          padding: { top: 8, bottom: 8 },
        }}
      />
    </div>
  )
}

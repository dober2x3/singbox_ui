"use client"

import { useRef, useCallback } from "react"
import Editor, { OnMount } from "@monaco-editor/react"
import type { editor } from "monaco-editor"

/** Props for the JsonEditor component. */
interface JsonEditorProps {
  value: string
  onChange?: (value: string) => void
  readOnly?: boolean
  height?: string | number
}

/**
 * Monaco-based JSON editor with dark theme, line numbers, and formatting support.
 */
export function JsonEditor({ value, onChange, readOnly = false, height = "500px" }: JsonEditorProps) {
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null)

  /** Stores the Monaco editor instance on mount. */
  const handleEditorDidMount: OnMount = useCallback((editor) => {
    editorRef.current = editor
  }, [])

  /** Forwards editor value changes to the parent onChange handler. */
  const handleChange = useCallback((newValue: string | undefined) => {
    if (onChange && newValue !== undefined) {
      onChange(newValue)
    }
  }, [onChange])

  return (
    <div className="h-full border rounded-lg overflow-hidden bg-[#1e1e1e]">
      <Editor
        height={height}
        defaultLanguage="json"
        value={value}
        onChange={handleChange}
        onMount={handleEditorDidMount}
        theme="vs-dark"
        options={{
          readOnly,
          minimap: { enabled: false },
          fontSize: 13,
          lineNumbers: "on",
          scrollBeyondLastLine: false,
          automaticLayout: true,
          tabSize: 2,
          wordWrap: "on",
          folding: true,
          formatOnPaste: true,
          formatOnType: true,
        }}
      />
    </div>
  )
}

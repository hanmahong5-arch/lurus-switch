import { memo, useState, useCallback, type ReactNode } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import { Copy, Check, ExternalLink } from 'lucide-react'
import { cn } from '../../lib/utils'

interface Props {
  content: string
  className?: string
}

// MarkdownBody renders a message body as rich markdown. Memoized by
// `content` so a 500-msg conversation doesn't reparse on every re-render.
function MarkdownBodyImpl({ content, className }: Props) {
  return (
    <div className={cn('markdown-body text-sm text-foreground/90 leading-relaxed', className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[[rehypeHighlight, { detect: true, ignoreMissing: true }]]}
        components={{
          h1: ({ children }) => (
            <h1 className="mt-3 mb-2 text-lg font-semibold border-l-2 border-primary pl-2">{children}</h1>
          ),
          h2: ({ children }) => (
            <h2 className="mt-3 mb-2 text-base font-semibold border-l-2 border-primary/70 pl-2">{children}</h2>
          ),
          h3: ({ children }) => (
            <h3 className="mt-2 mb-1.5 text-sm font-semibold border-l-2 border-primary/50 pl-2">{children}</h3>
          ),
          p: ({ children }) => <p className="my-1.5 whitespace-pre-wrap break-words">{children}</p>,
          ul: ({ children }) => <ul className="list-disc pl-5 my-1.5 space-y-0.5">{children}</ul>,
          ol: ({ children }) => <ol className="list-decimal pl-5 my-1.5 space-y-0.5">{children}</ol>,
          li: ({ children }) => <li className="leading-relaxed">{children}</li>,
          blockquote: ({ children }) => (
            <blockquote className="border-l-2 border-border pl-3 my-2 italic text-muted-foreground">
              {children}
            </blockquote>
          ),
          a: ({ href, children }) => (
            <a
              href={href}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-0.5 underline text-primary hover:text-primary/80"
            >
              {children}
              <ExternalLink className="h-3 w-3 inline-block" />
            </a>
          ),
          table: ({ children }) => (
            <div className="my-2 overflow-x-auto rounded border border-border">
              <table className="w-full text-xs">{children}</table>
            </div>
          ),
          thead: ({ children }) => <thead className="bg-muted/50">{children}</thead>,
          th: ({ children }) => <th className="px-2 py-1 text-left font-semibold border-b border-border">{children}</th>,
          tr: ({ children }) => <tr className="even:bg-muted/20">{children}</tr>,
          td: ({ children }) => <td className="px-2 py-1 border-b border-border/40">{children}</td>,
          hr: () => <hr className="my-3 border-border" />,
          code: ({ className: cls, children, ...rest }) => {
            const text = childrenToString(children)
            const isBlock = !!cls && cls.includes('language-')
            if (!isBlock) {
              return (
                <code className="px-1 py-0.5 rounded bg-muted text-foreground font-mono text-[0.85em]" {...rest}>
                  {children}
                </code>
              )
            }
            const lang = (cls?.match(/language-([\w-]+)/) || [])[1] || ''
            return <CodeBlock lang={lang} code={text} highlightClass={cls} />
          },
          pre: ({ children }) => <>{children}</>,
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}

function childrenToString(children: ReactNode): string {
  if (typeof children === 'string') return children
  if (Array.isArray(children)) return children.map(childrenToString).join('')
  if (children && typeof children === 'object' && 'props' in children) {
    const props = (children as { props?: { children?: ReactNode } }).props
    return childrenToString(props?.children ?? '')
  }
  return ''
}

function CodeBlock({ lang, code, highlightClass }: { lang: string; code: string; highlightClass?: string }) {
  const [copied, setCopied] = useState(false)
  const onCopy = useCallback(() => {
    void navigator.clipboard.writeText(code).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1200)
    })
  }, [code])

  return (
    <div className="my-2 rounded border border-border overflow-hidden bg-[#0d1117]">
      <div className="flex items-center justify-between px-2 py-1 bg-muted/40 text-[10px] uppercase tracking-wider text-muted-foreground font-mono">
        <span>{lang || 'text'}</span>
        <button
          onClick={onCopy}
          className="inline-flex items-center gap-1 hover:text-foreground transition-colors"
          title="Copy code"
        >
          {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
          {copied ? 'copied' : 'copy'}
        </button>
      </div>
      <pre className="overflow-x-auto p-3 text-xs leading-relaxed">
        <code className={cn('hljs font-mono', highlightClass)}>{code}</code>
      </pre>
    </div>
  )
}

export const MarkdownBody = memo(MarkdownBodyImpl, (a, b) => a.content === b.content && a.className === b.className)

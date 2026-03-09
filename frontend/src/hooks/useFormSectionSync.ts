import { useEffect, useCallback } from 'react'
import { sectionDomId } from '../lib/toolSchema'
import type { SectionDescriptor } from '../lib/toolSchema'

/** Syncs IntersectionObserver with the ContextSidebar active section. */
export function useFormSectionSync(
  toolId: string,
  sections: SectionDescriptor[],
  setActiveSection: (s: string) => void,
): { scrollToSection: (sectionId: string) => void } {
  useEffect(() => {
    if (sections.length === 0) return

    const observer = new IntersectionObserver(
      (entries) => {
        // Find the topmost visible section
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)

        if (visible.length > 0) {
          const id = visible[0].target.id
          // id format: ${toolId}-section-${sectionId}
          const prefix = `${toolId}-section-`
          if (id.startsWith(prefix)) {
            setActiveSection(id.slice(prefix.length))
          }
        }
      },
      { threshold: 0, rootMargin: '-40% 0px -55% 0px' },
    )

    sections.forEach(({ id }) => {
      const el = document.getElementById(sectionDomId(toolId, id))
      if (el) observer.observe(el)
    })

    return () => observer.disconnect()
  }, [toolId, sections, setActiveSection])

  const scrollToSection = useCallback(
    (sectionId: string) => {
      const el = document.getElementById(sectionDomId(toolId, sectionId))
      if (el) {
        el.scrollIntoView({ behavior: 'smooth', block: 'start' })
        setActiveSection(sectionId)
      }
    },
    [toolId, setActiveSection],
  )

  return { scrollToSection }
}

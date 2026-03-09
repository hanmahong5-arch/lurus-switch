export interface SectionDescriptor {
  id: string
  titleKey: string
  descKey: string
  icon: string
}

export interface ToolSchema {
  toolId: string
  version: string
  updatedAt: string
  sections: SectionDescriptor[]
}

/** Returns the DOM id used for a section element. */
export function sectionDomId(toolId: string, sectionId: string): string {
  return `${toolId}-section-${sectionId}`
}

/** Returns the sections for a given tool from a schema list. */
export function getToolSections(toolId: string, schemas: ToolSchema[]): SectionDescriptor[] {
  return schemas.find((s) => s.toolId === toolId)?.sections ?? []
}

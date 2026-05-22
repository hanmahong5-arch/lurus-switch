import { create } from 'zustand'
import {
  ListDepartments, GetDepartmentTree, CreateDepartment, UpdateDepartment, DeleteDepartment,
  ListEmployees, CreateEmployee, UpdateEmployee, DeactivateEmployee, FindEmployeeByEmail,
  ImportEmployeesCSV,
} from '../../wailsjs/go/main/App'

// Mirrors of internal/orgsync Go types. The Role union covers the 6
// roles defined in orgsync.go — keep in sync if a new role lands.

export type OrgRole =
  | 'employee' | 'team_lead' | 'dept_admin'
  | 'it_admin' | 'compliance' | 'finance'

export interface Department {
  id: string
  parentId: string
  name: string
  costCenter: string
  createdAt: string
  updatedAt: string
}

export interface Employee {
  id: string
  ssoSubject: string
  email: string
  displayName: string
  departmentId: string
  role: OrgRole
  managerId: string
  active: boolean
  createdAt: string
  updatedAt: string
}

export interface DepartmentTreeNode {
  department: Department
  employeeCount: number
  children: DepartmentTreeNode[]
}

interface State {
  departments: Department[]
  tree: DepartmentTreeNode[]
  employees: Employee[]
  selectedDeptId: string
  showInactive: boolean
  loading: boolean
  error: string | null

  load: () => Promise<void>
  loadEmployees: (deptId: string, includeInactive: boolean) => Promise<void>
  selectDept: (deptId: string) => void
  toggleInactive: () => void
  createDept: (d: Partial<Department>) => Promise<Department | null>
  updateDept: (d: Partial<Department>) => Promise<Department | null>
  deleteDept: (id: string) => Promise<void>
  createEmployee: (e: Partial<Employee>) => Promise<Employee | null>
  updateEmployee: (e: Partial<Employee>) => Promise<Employee | null>
  deactivateEmployee: (id: string) => Promise<void>
  findByEmail: (email: string) => Promise<Employee | null>
  importCSV: (content: string, defaultDeptId: string) => Promise<CSVImportResult | null>
}

export interface CSVImportError {
  lineNumber: number
  email: string
  reason: string
}

export interface CSVImportResult {
  created: number
  updated: number
  skipped: number
  errorRows: CSVImportError[]
}

export const useOrgChartStore = create<State>((set, get) => ({
  departments: [],
  tree: [],
  employees: [],
  selectedDeptId: '',
  showInactive: false,
  loading: false,
  error: null,

  load: async () => {
    set({ loading: true, error: null })
    try {
      const [departments, tree] = await Promise.all([
        ListDepartments(),
        GetDepartmentTree(),
      ])
      set({
        departments: (departments || []) as unknown as Department[],
        tree: (tree || []) as unknown as DepartmentTreeNode[],
        loading: false,
      })
      const sel = get().selectedDeptId
      if (sel) {
        await get().loadEmployees(sel, get().showInactive)
      }
    } catch (e: any) {
      set({ error: e?.message ?? String(e), loading: false })
    }
  },

  loadEmployees: async (deptId, includeInactive) => {
    try {
      const emps = await ListEmployees(deptId, !includeInactive)
      set({ employees: (emps || []) as unknown as Employee[] })
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
    }
  },

  selectDept: (deptId) => {
    set({ selectedDeptId: deptId })
    void get().loadEmployees(deptId, get().showInactive)
  },

  toggleInactive: () => {
    const next = !get().showInactive
    set({ showInactive: next })
    if (get().selectedDeptId) {
      void get().loadEmployees(get().selectedDeptId, next)
    }
  },

  createDept: async (d) => {
    try {
      const r = await CreateDepartment(d as any)
      await get().load()
      return r as unknown as Department
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
      return null
    }
  },

  updateDept: async (d) => {
    try {
      const r = await UpdateDepartment(d as any)
      await get().load()
      return r as unknown as Department
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
      return null
    }
  },

  deleteDept: async (id) => {
    try {
      await DeleteDepartment(id)
      await get().load()
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
    }
  },

  createEmployee: async (e) => {
    try {
      const r = await CreateEmployee(e as any)
      await get().loadEmployees(get().selectedDeptId, get().showInactive)
      return r as unknown as Employee
    } catch (err: any) {
      set({ error: err?.message ?? String(err) })
      return null
    }
  },

  updateEmployee: async (e) => {
    try {
      const r = await UpdateEmployee(e as any)
      await get().loadEmployees(get().selectedDeptId, get().showInactive)
      return r as unknown as Employee
    } catch (err: any) {
      set({ error: err?.message ?? String(err) })
      return null
    }
  },

  deactivateEmployee: async (id) => {
    try {
      await DeactivateEmployee(id)
      await get().loadEmployees(get().selectedDeptId, get().showInactive)
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
    }
  },

  findByEmail: async (email) => {
    try {
      const r = await FindEmployeeByEmail(email)
      return r as unknown as Employee
    } catch {
      return null
    }
  },

  importCSV: async (content, defaultDeptId) => {
    try {
      const r = await ImportEmployeesCSV(content, defaultDeptId)
      await get().load()
      return r as unknown as CSVImportResult
    } catch (e: any) {
      set({ error: e?.message ?? String(e) })
      return null
    }
  },
}))

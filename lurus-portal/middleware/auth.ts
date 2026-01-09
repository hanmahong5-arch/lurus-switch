// Auth middleware - protects dashboard routes
export default defineNuxtRouteMiddleware(async (to) => {
  const user = useSupabaseUser()

  // If not authenticated and trying to access protected route
  if (!user.value && to.path.startsWith('/dashboard')) {
    return navigateTo('/auth/login', {
      query: { redirect: to.fullPath }
    })
  }
})

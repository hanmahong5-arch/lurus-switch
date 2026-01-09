// Guest middleware - redirects authenticated users away from auth pages
export default defineNuxtRouteMiddleware(async () => {
  const user = useSupabaseUser()

  // If authenticated and trying to access auth pages
  if (user.value) {
    return navigateTo('/dashboard')
  }
})

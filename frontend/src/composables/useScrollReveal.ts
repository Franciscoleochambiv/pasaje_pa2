import { onMounted, onUnmounted } from 'vue'

export function useScrollReveal() {
  let observer: IntersectionObserver | null = null

  onMounted(() => {
    // Small delay to ensure DOM is fully rendered
    setTimeout(() => {
      const elements = document.querySelectorAll('.reveal')
      if (elements.length === 0) return

      observer = new IntersectionObserver(
        (entries) => {
          entries.forEach((entry) => {
            if (entry.isIntersecting) {
              entry.target.classList.add('is-revealed')
              observer?.unobserve(entry.target)
            }
          })
        },
        { threshold: 0.05, rootMargin: '0px 0px -20px 0px' }
      )

      elements.forEach((el) => observer?.observe(el))
    }, 100)
  })

  onUnmounted(() => {
    observer?.disconnect()
  })

  // Return dummy function for template compatibility
  return { reveal: () => {} }
}

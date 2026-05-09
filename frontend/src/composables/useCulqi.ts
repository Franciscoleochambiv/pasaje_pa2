import { ref } from 'vue'

declare global {
  interface Window {
    Culqi: any
    culqi: () => void
  }
}

const apiBase = import.meta.env.VITE_API_URL ?? ''

export function useCulqi() {
  const loading = ref(false)
  const error = ref('')

  async function initCulqi(): Promise<void> {
    const res = await fetch(`${apiBase}/api/payment/config`)
    if (!res.ok) throw new Error('No se pudo obtener la configuracion de pago')
    const { public_key } = await res.json()
    if (window.Culqi) {
      window.Culqi.publicKey = public_key
    } else {
      throw new Error('Culqi Checkout no esta cargado. Recarga la pagina.')
    }
  }

  function openCheckout(options: {
    title: string
    currency: string
    amount: number
    description: string
  }): Promise<string> {
    return new Promise((resolve, reject) => {
      if (!window.Culqi) {
        reject(new Error('Culqi Checkout no esta disponible'))
        return
      }

      let settled = false

      window.Culqi.settings({
        title: options.title,
        currency: options.currency,
        amount: options.amount,
        description: options.description,
      })

      window.Culqi.options({
        lang: 'es',
        installments: false,
        paymentMethods: {
          tarjeta: true,
          yape: true,
          bancaMovil: false,
          agente: false,
          billetera: false,
          cuotealo: false,
        },
        style: {
          logo: '',
          bannerColor: '#1b55f5',
          buttonBackground: '#1b55f5',
          buttonText: 'Pagar',
          buttonTextColor: '#ffffff',
          menuColor: '#1b55f5',
          priceColor: '#1b55f5',
        },
      })

      // Callback cuando Culqi genera token o error
      window.culqi = () => {
        if (settled) return
        settled = true
        if (window.Culqi.token) {
          resolve(window.Culqi.token.id)
        } else if (window.Culqi.error) {
          reject(new Error(window.Culqi.error.user_message || 'Error en el pago'))
        }
      }

      // Detectar cierre del modal (usuario cancela)
      // Culqi no tiene evento de cierre oficial, usamos MutationObserver
      const observer = new MutationObserver(() => {
        // El iframe de Culqi se remueve del DOM al cerrar
        const culqiFrame = document.getElementById('culqi-checkout')
          || document.querySelector('iframe[src*="culqi"]')
          || document.querySelector('.culqi-checkout-modal')

        if (!culqiFrame && settled === false) {
          // Esperar 500ms para confirmar que realmente se cerró (no es un render intermedio)
          setTimeout(() => {
            const stillOpen = document.getElementById('culqi-checkout')
              || document.querySelector('iframe[src*="culqi"]')
            if (!stillOpen && !settled) {
              settled = true
              observer.disconnect()
              reject(new Error('Pago cancelado'))
            }
          }, 500)
        }
      })

      // Observar cambios en el body para detectar cierre
      setTimeout(() => {
        observer.observe(document.body, { childList: true, subtree: true })
      }, 1000) // esperar que el modal se abra primero

      // Timeout de seguridad: si después de 2 minutos no hay respuesta, cancelar
      setTimeout(() => {
        if (!settled) {
          settled = true
          observer.disconnect()
          reject(new Error('Tiempo de espera agotado. Intenta nuevamente.'))
        }
      }, 120000) // 2 minutos

      window.Culqi.open()
    })
  }

  function closeCheckout() {
    if (window.Culqi) {
      try { window.Culqi.close() } catch { /* ignore */ }
    }
  }

  return { initCulqi, openCheckout, closeCheckout, loading, error }
}

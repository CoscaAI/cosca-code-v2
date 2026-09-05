import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { TabBar } from './TabBar'

describe('TabBar', () => {
  it('renderiza as abas abertas', () => {
    render(
      <TabBar
        files={['src/main.go', 'src/util.go']}
        active="src/main.go"
        onActivate={() => {}}
        onClose={() => {}}
      />,
    )
    expect(screen.getByText('main.go')).toBeTruthy()
    expect(screen.getByText('util.go')).toBeTruthy()
  })

  it('chama onActivate ao clicar na aba', () => {
    const onActivate = vi.fn()
    render(
      <TabBar
        files={['a.go']}
        active="a.go"
        onActivate={onActivate}
        onClose={() => {}}
      />,
    )
    fireEvent.click(screen.getByText('a.go'))
    expect(onActivate).toHaveBeenCalledWith('a.go')
  })

  it('chama onClose ao clicar no ✕ (sem ativar)', () => {
    const onClose = vi.fn()
    const onActivate = vi.fn()
    render(
      <TabBar
        files={['a.go']}
        active="a.go"
        onActivate={onActivate}
        onClose={onClose}
      />,
    )
    fireEvent.click(screen.getByRole('button'))
    expect(onClose).toHaveBeenCalledWith('a.go')
    expect(onActivate).not.toHaveBeenCalled()
  })
})

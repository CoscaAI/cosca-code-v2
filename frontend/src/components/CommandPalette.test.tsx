import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { CommandPalette } from './CommandPalette'

describe('CommandPalette', () => {
  const commands = [
    { id: 'search', title: 'Search', run: vi.fn() },
    { id: 'git', title: 'Source Control', run: vi.fn() },
  ]

  it('renderiza e filtra os comandos', () => {
    render(<CommandPalette commands={commands} onClose={() => {}} />)
    fireEvent.change(screen.getByPlaceholderText('comando, símbolo ou linguagem natural…'), { target: { value: 'source' } })
    expect(screen.getByText('Source Control')).toBeTruthy()
    expect(screen.queryByText('Search')).toBeFalsy()
  })

  it('executa o comando ao clicar', () => {
    render(<CommandPalette commands={commands} onClose={() => {}} />)
    fireEvent.click(screen.getByText('Search'))
    expect(commands[0].run).toHaveBeenCalled()
  })
})

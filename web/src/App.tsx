import { useState, useEffect } from 'react'

interface Account {
  id: string
  bank_name: string
  account_number: string
  account_type: string
  balance: number
  currency: string
  is_active: boolean
}

interface Transaction {
  id: string
  type: string
  amount: number
  description: string
  merchant: string
  category: string
  channel: string
  transaction_date: string
}

interface Overview {
  total_balance: number
  total_accounts: number
  total_debit: number
  total_credit: number
  transaction_count: number
}

interface CategorySpend {
  category: string
  amount: number
  count: number
}

interface BrainMetrics {
  accuracy: number
  total_predictions: number
  correct_predictions: number
  user_corrections: number
  training_size: number
}

function formatINR(amount: number): string {
  return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 }).format(amount)
}

function App() {
  const [tab, setTab] = useState<'overview' | 'transactions' | 'accounts' | 'brain'>('overview')
  const [overview, setOverview] = useState<Overview | null>(null)
  const [accounts, setAccounts] = useState<Account[]>([])
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [spend, setSpend] = useState<CategorySpend[]>([])
  const [brain, setBrain] = useState<BrainMetrics | null>(null)

  useEffect(() => {
    fetch('/api/overview').then(r => r.json()).then(setOverview).catch(() => {})
    fetch('/api/accounts').then(r => r.json()).then(setAccounts).catch(() => {})
    fetch('/api/transactions').then(r => r.json()).then(setTransactions).catch(() => {})
    fetch('/api/spend/categories').then(r => r.json()).then(setSpend).catch(() => {})
    fetch('/api/brain/status').then(r => r.json()).then(d => { if (d.accuracy !== undefined) setBrain(d) }).catch(() => {})
  }, [])

  const tabs = [
    { id: 'overview' as const, label: 'Overview' },
    { id: 'transactions' as const, label: 'Transactions' },
    { id: 'accounts' as const, label: 'Accounts' },
    { id: 'brain' as const, label: 'Brain' },
  ]

  return (
    <div className="min-h-screen p-6 max-w-6xl mx-auto">
      <header className="mb-8">
        <h1 className="text-3xl font-bold text-white">Finance Agent</h1>
        <p className="text-gray-400 mt-1">Personal financial intelligence dashboard</p>
      </header>

      <nav className="flex gap-1 mb-8 bg-gray-900 rounded-lg p-1">
        {tabs.map(t => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
              tab === t.id ? 'bg-indigo-600 text-white' : 'text-gray-400 hover:text-white hover:bg-gray-800'
            }`}
          >
            {t.label}
          </button>
        ))}
      </nav>

      {tab === 'overview' && <OverviewTab overview={overview} spend={spend} />}
      {tab === 'transactions' && <TransactionsTab transactions={transactions} />}
      {tab === 'accounts' && <AccountsTab accounts={accounts} />}
      {tab === 'brain' && <BrainTab brain={brain} />}
    </div>
  )
}

function OverviewTab({ overview, spend }: { overview: Overview | null; spend: CategorySpend[] }) {
  if (!overview) return <p className="text-gray-500">Loading...</p>

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="Total Balance" value={formatINR(overview.total_balance)} color="text-emerald-400" />
        <StatCard label="Monthly Debit" value={formatINR(overview.total_debit)} color="text-red-400" />
        <StatCard label="Monthly Credit" value={formatINR(overview.total_credit)} color="text-green-400" />
        <StatCard label="Transactions" value={String(overview.transaction_count)} color="text-blue-400" />
      </div>

      {spend.length > 0 && (
        <div className="bg-gray-900 rounded-xl p-6">
          <h3 className="text-lg font-semibold mb-4">Spend by Category</h3>
          <div className="space-y-3">
            {spend.sort((a, b) => b.amount - a.amount).map(s => {
              const maxAmount = spend[0]?.amount || 1
              return (
                <div key={s.category} className="flex items-center gap-3">
                  <span className="w-32 text-sm text-gray-400 truncate">{s.category}</span>
                  <div className="flex-1 bg-gray-800 rounded-full h-3 overflow-hidden">
                    <div className="bg-indigo-500 h-full rounded-full" style={{ width: `${(s.amount / maxAmount) * 100}%` }} />
                  </div>
                  <span className="text-sm font-mono w-24 text-right">{formatINR(s.amount)}</span>
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}

function StatCard({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div className="bg-gray-900 rounded-xl p-4">
      <p className="text-xs text-gray-500 uppercase tracking-wide">{label}</p>
      <p className={`text-2xl font-bold mt-1 ${color}`}>{value}</p>
    </div>
  )
}

function TransactionsTab({ transactions }: { transactions: Transaction[] }) {
  if (transactions.length === 0) return <p className="text-gray-500">No transactions yet. Run sync to fetch from email.</p>

  return (
    <div className="bg-gray-900 rounded-xl overflow-hidden">
      <table className="w-full">
        <thead className="bg-gray-800">
          <tr>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">Date</th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">Description</th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase">Category</th>
            <th className="px-4 py-3 text-right text-xs font-medium text-gray-400 uppercase">Amount</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-800">
          {transactions.map(txn => (
            <tr key={txn.id} className="hover:bg-gray-800/50">
              <td className="px-4 py-3 text-sm text-gray-400">{new Date(txn.transaction_date).toLocaleDateString('en-IN', { day: '2-digit', month: 'short' })}</td>
              <td className="px-4 py-3 text-sm">{txn.merchant || txn.description}</td>
              <td className="px-4 py-3"><span className="text-xs bg-gray-800 text-gray-300 px-2 py-1 rounded">{txn.category || '—'}</span></td>
              <td className={`px-4 py-3 text-sm text-right font-mono ${txn.type === 'debit' ? 'text-red-400' : 'text-green-400'}`}>
                {txn.type === 'debit' ? '-' : '+'}{formatINR(txn.amount)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function AccountsTab({ accounts }: { accounts: Account[] }) {
  if (accounts.length === 0) return <p className="text-gray-500">No accounts detected yet.</p>

  return (
    <div className="grid gap-4 md:grid-cols-2">
      {accounts.map(acc => (
        <div key={acc.id} className="bg-gray-900 rounded-xl p-5">
          <div className="flex justify-between items-start">
            <div>
              <p className="font-semibold">{acc.bank_name}</p>
              <p className="text-sm text-gray-400">••{acc.account_number} · {acc.account_type}</p>
            </div>
            <span className={`text-xs px-2 py-1 rounded ${acc.is_active ? 'bg-emerald-900 text-emerald-300' : 'bg-gray-800 text-gray-500'}`}>
              {acc.is_active ? 'Active' : 'Inactive'}
            </span>
          </div>
          <p className="text-2xl font-bold text-emerald-400 mt-3">{formatINR(acc.balance)}</p>
        </div>
      ))}
    </div>
  )
}

function BrainTab({ brain }: { brain: BrainMetrics | null }) {
  if (!brain) return <p className="text-gray-500">No brain data yet. Train the model to see metrics.</p>

  const accuracy = (brain.accuracy * 100).toFixed(1)
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="Accuracy" value={`${accuracy}%`} color="text-indigo-400" />
        <StatCard label="Predictions" value={String(brain.total_predictions)} color="text-blue-400" />
        <StatCard label="Corrections" value={String(brain.user_corrections)} color="text-amber-400" />
        <StatCard label="Training Size" value={String(brain.training_size)} color="text-purple-400" />
      </div>

      <div className="bg-gray-900 rounded-xl p-6">
        <h3 className="text-lg font-semibold mb-4">Model Performance</h3>
        <div className="flex items-center gap-4">
          <div className="flex-1 bg-gray-800 rounded-full h-4 overflow-hidden">
            <div className="bg-indigo-500 h-full rounded-full transition-all" style={{ width: `${brain.accuracy * 100}%` }} />
          </div>
          <span className="text-sm font-mono">{accuracy}%</span>
        </div>
        <p className="text-xs text-gray-500 mt-2">
          {brain.correct_predictions}/{brain.total_predictions} correct predictions
        </p>
      </div>
    </div>
  )
}

export default App

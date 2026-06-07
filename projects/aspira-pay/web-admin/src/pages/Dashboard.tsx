import { useEffect, useState, useRef, useCallback } from 'react'
import { api, ensureAuth } from '../api/client'
import StatsCard from '../components/StatsCard'

const POLL_INTERVAL = 3000 // 3 seconds

export default function Dashboard() {
  const [stats, setStats] = useState<any>(null)
  const [error, setError] = useState('')
  const [authChecked, setAuthChecked] = useState(false)
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null)
  const [tick, setTick] = useState(0) // force re-render counter
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchDashboard = useCallback(async () => {
    try {
      const data = await api.getDashboard()
      setStats(data)
      setLastUpdate(new Date())
      setError('')
    } catch (err: any) {
      // Don't overwrite existing data on transient errors
      if (!stats) {
        setError(err.message)
      }
    }
  }, [stats])

  useEffect(() => {
    async function init() {
      try {
        await ensureAuth()
        setAuthChecked(true)
        await fetchDashboard()
      } catch (err: any) {
        setError(err.message)
      }
    }
    init()

    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [])

  // Start polling after auth is confirmed
  useEffect(() => {
    if (!authChecked) return

    intervalRef.current = setInterval(() => {
      fetchDashboard()
      setTick(t => t + 1)
    }, POLL_INTERVAL)

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
    }
  }, [authChecked, fetchDashboard])

  if (error && !stats) {
    return (
      <div className="bg-red-900/30 border border-red-800 rounded-lg p-4 text-red-400">
        Cannot connect to API: {error}
        <p className="text-sm mt-2">Make sure the API server is running on port 8080.</p>
      </div>
    )
  }

  if (!authChecked || !stats) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500 text-lg">
          <span className="inline-block animate-spin mr-3">⟳</span>
          Connecting to Aspira Pay V2...
        </div>
      </div>
    )
  }

  const formatTime = (d: Date) =>
    d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' })

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold">Dashboard</h2>
        <div className="flex items-center gap-2 text-xs text-gray-500">
          <span className="inline-block w-2 h-2 rounded-full bg-green-400 animate-pulse" />
          Live &middot; updated {lastUpdate ? formatTime(lastUpdate) : '--'}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <StatsCard
          title="Total Payments"
          value={stats?.total_payments?.toLocaleString() || '0'}
          icon="💱"
          subtitle="all-time"
        />
        <StatsCard
          title="Total Users"
          value={stats?.total_users?.toLocaleString() || '0'}
          icon="👥"
          subtitle="registered"
        />
        <StatsCard
          title="Settlement Batches"
          value={stats?.total_settlement_batches?.toLocaleString() || '0'}
          icon="📒"
          subtitle="completed"
        />
        <StatsCard
          title="System Status"
          value={stats?.system_status || 'Unknown'}
          icon="🟢"
          subtitle={stats?.engine_status || 'connected'}
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h3 className="text-lg font-semibold mb-4">Quick Actions</h3>
          <div className="space-y-3">
            <a href="/transactions" className="block p-3 bg-gray-800 rounded-lg hover:bg-gray-700 transition-colors">
              💱 View Transactions
            </a>
            <a href="/users" className="block p-3 bg-gray-800 rounded-lg hover:bg-gray-700 transition-colors">
              👥 Manage Users
            </a>
            <a href="/audit" className="block p-3 bg-gray-800 rounded-lg hover:bg-gray-700 transition-colors">
              ⛓️ Blockchain Audit Explorer
            </a>
          </div>
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
          <h3 className="text-lg font-semibold mb-4">System Info</h3>
          <dl className="space-y-2 text-sm">
            <div className="flex justify-between">
              <dt className="text-gray-500">Version</dt>
              <dd>2.0.0-sandbox</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">FX Rates Source</dt>
              <dd className="text-green-400">Frankfurter API (live)</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Settlement Base</dt>
              <dd className="text-blue-400">USD</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Poll Interval</dt>
              <dd className="text-gray-400">3s</dd>
            </div>
          </dl>
        </div>
      </div>
    </div>
  )
}

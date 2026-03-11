import { useState, useEffect, useRef } from 'react'
import {
  Search,
  Home,
  CheckCircle,
  FileEdit,
  UploadCloud,
  XCircle,
  BarChart2,
  MapPin,
  BedDouble,
  Bath,
  Maximize
} from 'lucide-react'

// Replace with empty string if serving over same origin, otherwise localhost
const API_URL = 'http://localhost:8080/api'

// --- Types ---
type PipelineStats = {
  total: number
  draft: number
  prepped: number
  approved: number
  posted: number
  failed: number
}

type Listing = {
  id: string
  title: string
  description?: string
  property_type: string
  listing_type: string
  price?: number
  price_currency: string
  status: string
  tags: string[]
  content_languages: string[]
  district?: string
  province?: string
  bedrooms?: number
  bathrooms?: number
  area_sqm?: number
  similarity_score?: number
}

// --- Icons Mapping ---
const statusIcons: Record<string, React.ReactNode> = {
  total: <BarChart2 size={16} />,
  draft: <FileEdit size={16} />,
  prepped: <CheckCircle size={16} className="text-accent-amber" />,
  approved: <CheckCircle size={16} className="text-accent-cyan" />,
  posted: <UploadCloud size={16} className="text-accent-emerald" />,
  failed: <XCircle size={16} className="text-accent-rose" />
}

const statusColors: Record<string, string> = {
  draft: 'bg-text-secondary/15 text-text-secondary',
  prepped: 'bg-accent-amber/15 text-accent-amber',
  approved: 'bg-accent-cyan/15 text-accent-cyan',
  posted: 'bg-accent-emerald/15 text-accent-emerald',
  failed: 'bg-accent-rose/15 text-accent-rose'
}

export default function App() {
  const [health, setHealth] = useState({ status: 'connecting', listings: 0 })
  const [stats, setStats] = useState<PipelineStats | null>(null)
  const [listings, setListings] = useState<Listing[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState<Listing[]>([])
  const [isSearching, setIsSearching] = useState(false)

  const searchInputRef = useRef<HTMLInputElement>(null)

  // CMD+K Shortcut
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        searchInputRef.current?.focus()
      }
      if (e.key === 'Escape') {
        setIsSearching(false)
        searchInputRef.current?.blur()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  // Initial Fetch
  useEffect(() => {
    fetchHealth()
    fetchPipeline()
    fetchListings()
    const interval = setInterval(fetchHealth, 10000)
    return () => clearInterval(interval)
  }, [])

  // Search Debounce
  useEffect(() => {
    if (searchQuery.trim().length < 2) {
      setSearchResults([])
      setIsSearching(false)
      return
    }
    const timer = setTimeout(async () => {
      setIsSearching(true)
      try {
        const res = await fetch(`${API_URL}/search`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ query: searchQuery, limit: 5 })
        })
        const data = await res.json()
        setSearchResults(data.results || [])
      } catch (err) {
        console.error("Search failed", err)
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [searchQuery])

  // Fetch logic
  const fetchHealth = async () => {
    try {
      const res = await fetch(`${API_URL}/health`)
      const data = await res.json()
      setHealth({ status: data.status, listings: data.listings })
    } catch {
      setHealth({ status: 'offline', listings: 0 })
    }
  }

  const fetchPipeline = async () => {
    try {
      const res = await fetch(`${API_URL}/pipeline`)
      setStats(await res.json())
    } catch (e) { console.error(e) }
  }

  const fetchListings = async () => {
    try {
      const res = await fetch(`${API_URL}/listings`)
      const data = await res.json()
      setListings(data.listings || [])
    } catch (e) { console.error(e) }
  }

  return (
    <div className="min-h-screen text-text-primary overflow-x-hidden font-inter">
      {/* Animated background */}
      <div className="bg-grid fixed inset-0 z-0 pointer-events-none" />

      {/* Header */}
      <header className="sticky top-0 z-50 py-4 px-6 border-b border-border-subtle bg-bg-primary/80 backdrop-blur-xl flex justify-between items-center">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 flex items-center justify-center rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 text-lg shadow-lg shadow-indigo-500/20">
            <Home size={20} className="text-white" />
          </div>
          <h1 className="font-bold text-lg tracking-tight gradient-hero-text">Super-Agent</h1>
          <span className="text-[11px] font-medium px-2 py-0.5 rounded-full border border-border-subtle text-text-muted">v0.1.0</span>
        </div>

        <div className="flex items-center gap-2 text-sm text-text-secondary">
          <div className={`w-2 h-2 rounded-full ${health.status === 'healthy' ? 'bg-accent-emerald animate-pulse' : 'bg-accent-rose'}`} />
          {health.status === 'healthy' ? `Connected · ${health.listings} properties` : 'Disconnected'}
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-10 relative z-10">

        {/* Spotlight Search */}
        <section className="mb-12">
          <div className="relative max-w-2xl mx-auto">
            <div className="relative flex items-center">
              <Search className="absolute left-4 text-text-muted" size={20} />
              <input
                ref={searchInputRef}
                type="text"
                placeholder="Search listings... (e.g., 'condo near BTS Ari under 20k')"
                className="w-full bg-bg-card border border-border-subtle focus:border-indigo-500/50 focus:ring-4 focus:ring-indigo-500/10 rounded-2xl py-4 pl-12 pr-16 outline-none transition-all placeholder:text-text-muted"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onBlur={() => setTimeout(() => setIsSearching(false), 200)}
                onFocus={() => { if (searchQuery.length >= 2) setIsSearching(true) }}
              />
              <div className="absolute right-4 flex items-center gap-1">
                <kbd className="px-1.5 py-0.5 rounded text-[11px] font-mono bg-white/5 border border-border-subtle text-text-muted">⌘</kbd>
                <kbd className="px-1.5 py-0.5 rounded text-[11px] font-mono bg-white/5 border border-border-subtle text-text-muted">K</kbd>
              </div>
            </div>

            {/* Search Results Dropdown */}
            {isSearching && (
              <div className="absolute top-full mt-2 w-full bg-bg-secondary border border-border-subtle rounded-xl overflow-hidden shadow-2xl shadow-indigo-500/10 backdrop-blur-lg">
                {searchResults.length > 0 ? (
                  searchResults.map(res => (
                    <div key={res.id} className="px-4 py-3 hover:bg-bg-card-hover border-b border-border-subtle last:border-0 cursor-pointer transition-colors">
                      <div className="font-medium">{res.title}</div>
                      <div className="flex gap-3 text-xs text-text-secondary mt-1">
                        <span className="capitalize">{res.property_type}</span>
                        {res.price && <span className="text-text-primary">฿{res.price.toLocaleString()}</span>}
                        {res.district && <span className="flex items-center gap-1"><MapPin size={12} />{res.district}</span>}
                        <span className={`px-1.5 py-0.5 rounded uppercase text-[9px] font-bold tracking-wider ${statusColors[res.status]}`}>{res.status}</span>
                      </div>
                    </div>
                  ))
                ) : (
                  <div className="px-4 py-6 text-center text-text-muted text-sm">
                    No results found for "{searchQuery}"
                  </div>
                )}
              </div>
            )}
          </div>
        </section>

        {/* Pipeline Overview */}
        <section className="mb-12">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-bold flex items-center gap-2 pt-2">
              <BarChart2 size={20} className="text-accent-indigo" />
              Pipeline Overview
            </h2>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
            {['total', 'draft', 'prepped', 'approved', 'posted', 'failed'].map((key) => {
              const count = stats ? stats[key as keyof PipelineStats] || 0 : 0
              return (
                <div key={key} className="bg-bg-card border border-border-subtle rounded-xl p-5 hover:bg-bg-card-hover hover:border-indigo-500/30 transition-all cursor-default">
                  <div className="text-[11px] uppercase tracking-wider font-semibold text-text-muted mb-2 flex items-center gap-1.5">
                    {statusIcons[key]} {key}
                  </div>
                  <div className={`text-3xl font-extrabold ${key === 'total' ? 'gradient-hero-text' : 'text-text-primary'}`}>
                    {stats ? count : '-'}
                  </div>
                </div>
              )
            })}
          </div>
        </section>

        {/* Listings Grid */}
        <section>
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-lg font-bold flex items-center gap-2">
              <Home size={20} className="text-accent-violet" />
              Recent Listings
              <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-indigo-500/15 text-accent-indigo">{listings.length}</span>
            </h2>
          </div>

          {listings.length === 0 ? (
            <div className="text-center py-20 bg-bg-card border border-border-subtle rounded-2xl">
              <div className="text-4xl mb-4">🏗️</div>
              <div className="text-text-secondary text-lg font-medium mb-1">No listings yet</div>
              <div className="text-text-muted text-sm max-w-md mx-auto">
                Sync your first listings from your database using the CLI to see them appear here automatically.
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {listings.map(l => {
                const price = l.price ? `฿${l.price.toLocaleString()}` : 'Price TBD'
                const unit = l.listing_type === 'rent' ? '/mo' : ''

                return (
                  <div key={l.id} className="bg-bg-card border border-border-subtle rounded-2xl overflow-hidden hover:-translate-y-1 hover:border-indigo-500/30 hover:shadow-2xl hover:shadow-indigo-500/10 transition-all duration-300 flex flex-col">
                    <div className="p-5 flex-1">
                      <div className="flex justify-between items-start gap-4 mb-3">
                        <h3 className="font-semibold text-[15px] leading-tight flex-1 line-clamp-2">{l.title}</h3>
                        <span className={`text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full whitespace-nowrap ${statusColors[l.status]}`}>
                          {l.status}
                        </span>
                      </div>

                      {l.description && (
                        <p className="text-[13px] text-text-secondary line-clamp-2 mb-4 leading-relaxed">
                          {l.description}
                        </p>
                      )}

                      {(l.tags && l.tags.length > 0) && (
                        <div className="flex flex-wrap gap-1.5 mb-4">
                          {l.tags.map(t => (
                            <span key={t} className="text-[10px] font-medium px-1.5 py-0.5 rounded border border-indigo-500/20 text-accent-indigo bg-indigo-500/5">
                              #{t}
                            </span>
                          ))}
                        </div>
                      )}

                      <div className="flex items-center gap-4 text-xs text-text-secondary mb-4">
                        {l.bedrooms !== undefined && l.bedrooms > 0 && <span className="flex items-center gap-1.5"><BedDouble size={14} /> {l.bedrooms}</span>}
                        {l.bathrooms !== undefined && l.bathrooms > 0 && <span className="flex items-center gap-1.5"><Bath size={14} /> {l.bathrooms}</span>}
                        {l.area_sqm !== undefined && l.area_sqm > 0 && <span className="flex items-center gap-1.5"><Maximize size={14} /> {l.area_sqm}m²</span>}
                      </div>

                    </div>

                    <div className="p-5 pt-0 mt-auto">
                      <div className="flex border-t border-border-subtle pt-4 justify-between items-end mb-4">
                        <div>
                          <span className="text-lg font-bold gradient-hero-text">{price}</span>
                          <span className="text-xs text-text-muted">{unit}</span>
                        </div>
                        <div className="flex gap-1">
                          {['th', 'en', 'my'].map(lang => (
                            <span key={lang} className={`text-[9px] font-bold uppercase px-1.5 py-0.5 rounded ${l.content_languages?.includes(lang) ? 'bg-emerald-500/15 text-accent-emerald' : 'bg-white/5 text-text-muted border border-border-subtle'}`}>
                              {lang}
                            </span>
                          ))}
                        </div>
                      </div>

                      <div className="flex gap-2">
                        <button className="flex-1 py-2 rounded-lg text-xs font-semibold bg-amber-500/10 text-accent-amber hover:bg-amber-500/20 transition-colors border border-amber-500/20 border-b-amber-500/40">
                          🎨 Smart Prep
                        </button>
                        <button className="flex-1 py-2 rounded-lg text-xs font-semibold bg-emerald-500/10 text-accent-emerald hover:bg-emerald-500/20 transition-colors border border-emerald-500/20 border-b-emerald-500/40">
                          📤 Sniper Post
                        </button>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </section>

      </main>
    </div>
  )
}

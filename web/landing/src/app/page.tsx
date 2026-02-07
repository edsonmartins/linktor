import { Navbar, Hero, Features, Channels, SDKs, Docs, CTA, Footer } from '@/components'

export default function Home() {
  return (
    <>
      <Navbar />
      <main>
        <Hero />
        <Features />
        <Channels />
        <SDKs />
        <Docs />
        <CTA />
      </main>
      <Footer />
    </>
  )
}

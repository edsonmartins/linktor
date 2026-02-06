import { Navbar, Hero, Features, Channels, SDKs, CTA, Footer } from '@/components'

export default function Home() {
  return (
    <>
      <Navbar />
      <main>
        <Hero />
        <Features />
        <Channels />
        <SDKs />
        <CTA />
      </main>
      <Footer />
    </>
  )
}

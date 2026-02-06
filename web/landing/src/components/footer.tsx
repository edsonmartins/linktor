'use client'

import Link from 'next/link'
import Image from 'next/image'
import { Github, Twitter, Linkedin, MessageCircle } from 'lucide-react'

const footerLinks = {
  product: {
    title: 'Product',
    links: [
      { label: 'Features', href: '#features' },
      { label: 'Channels', href: '#channels' },
      { label: 'Pricing', href: '/pricing' },
      { label: 'Changelog', href: '/changelog' },
    ],
  },
  developers: {
    title: 'Developers',
    links: [
      { label: 'Documentation', href: 'https://docs.linktor.io' },
      { label: 'API Reference', href: 'https://docs.linktor.io/api' },
      { label: 'SDKs', href: '#sdks' },
      { label: 'GitHub', href: 'https://github.com/linktor/linktor' },
    ],
  },
  resources: {
    title: 'Resources',
    links: [
      { label: 'Blog', href: '/blog' },
      { label: 'Guides', href: 'https://docs.linktor.io/guides' },
      { label: 'Examples', href: 'https://github.com/linktor/examples' },
      { label: 'Community', href: 'https://discord.gg/linktor' },
    ],
  },
  company: {
    title: 'Company',
    links: [
      { label: 'About', href: '/about' },
      { label: 'Contact', href: '/contact' },
      { label: 'Privacy', href: '/privacy' },
      { label: 'Terms', href: '/terms' },
    ],
  },
}

const socialLinks = [
  { icon: Github, href: 'https://github.com/linktor/linktor', label: 'GitHub' },
  { icon: Twitter, href: 'https://twitter.com/linktorhq', label: 'Twitter' },
  { icon: Linkedin, href: 'https://linkedin.com/company/linktor', label: 'LinkedIn' },
  { icon: MessageCircle, href: 'https://discord.gg/linktor', label: 'Discord' },
]

export function Footer() {
  return (
    <footer className="border-t border-border bg-card/50">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        {/* Main Footer */}
        <div className="py-12 grid grid-cols-2 md:grid-cols-6 gap-8">
          {/* Brand */}
          <div className="col-span-2">
            <Link href="/" className="inline-block">
              <Image
                src="/images/logo_fundo_escuro.png"
                alt="Linktor"
                width={180}
                height={50}
                className="h-12 w-auto"
              />
            </Link>
            <p className="mt-4 text-sm text-muted-foreground max-w-xs">
              Open source omnichannel conversation platform powered by AI.
              Connect all your messaging channels in one place.
            </p>
            {/* Social Links */}
            <div className="mt-6 flex gap-4">
              {socialLinks.map((social) => (
                <Link
                  key={social.label}
                  href={social.href}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-muted-foreground hover:text-primary transition-colors"
                  aria-label={social.label}
                >
                  <social.icon className="h-5 w-5" />
                </Link>
              ))}
            </div>
          </div>

          {/* Links */}
          {Object.values(footerLinks).map((section) => (
            <div key={section.title}>
              <h3 className="font-semibold text-foreground mb-4">{section.title}</h3>
              <ul className="space-y-3">
                {section.links.map((link) => (
                  <li key={link.label}>
                    <Link
                      href={link.href}
                      className="text-sm text-muted-foreground hover:text-primary transition-colors"
                    >
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* Bottom Bar */}
        <div className="py-6 border-t border-border flex flex-col sm:flex-row justify-between items-center gap-4">
          <p className="text-sm text-muted-foreground">
            &copy; {new Date().getFullYear()} Linktor. All rights reserved.
          </p>
          <p className="text-sm text-muted-foreground font-mono">
            Made with <span className="text-terminal-coral">&#x2665;</span> for developers
          </p>
        </div>
      </div>
    </footer>
  )
}

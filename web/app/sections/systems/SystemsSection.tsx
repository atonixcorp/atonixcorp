"use client";

export default function SystemsSection() {
  return (
    <section className="systems section wrap">
      <p className="section-index">05 / Integrated systems</p>

      <div className="system-grid">
        <article>
          <span className="system-symbol">◇</span>
          <h3>Defense-grade protection</h3>
          <p>
            Cyber-physical security, zero-trust architecture, autonomous threat
            detection, and secure hardware environments.
          </p>
        </article>

        <article>
          <span className="system-symbol">⌁</span>
          <h3>Developer ecosystem</h3>
          <p>
            A unified global network for multi-language programming, secure development,
            research hubs, and autonomous tooling.
          </p>
        </article>

        <article>
          <span className="system-symbol">⊕</span>
          <h3>Energy-driven computing</h3>
          <p>
            Solar-powered HPC, energy-aware cloud orchestration, efficient cooling,
            and autonomous energy management.
          </p>
        </article>
      </div>
    </section>
  );
}

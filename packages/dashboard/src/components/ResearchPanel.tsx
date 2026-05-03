import type { APIResearch } from '@ogame-bot/shared';
import { Show } from 'solid-js';

interface TechDef {
  key: keyof APIResearch;
  label: string;
}

interface ResearchCategory {
  name: string;
  techs: TechDef[];
}

const categories: ResearchCategory[] = [
  {
    name: 'Physics',
    techs: [
      { key: 'energyTechnology', label: 'Energy' },
      { key: 'laserTechnology', label: 'Laser' },
      { key: 'ionTechnology', label: 'Ion' },
      { key: 'hyperspaceTechnology', label: 'Hyperspace' },
      { key: 'plasmaTechnology', label: 'Plasma' },
    ],
  },
  {
    name: 'Drives',
    techs: [
      { key: 'combustionDrive', label: 'Combustion' },
      { key: 'impulseDrive', label: 'Impulse' },
      { key: 'hyperspaceDrive', label: 'Hyperspace' },
    ],
  },
  {
    name: 'Military',
    techs: [
      { key: 'weaponTechnology', label: 'Weapons' },
      { key: 'shieldingTechnology', label: 'Shielding' },
      { key: 'armourTechnology', label: 'Armor' },
    ],
  },
  {
    name: 'General',
    techs: [
      { key: 'espionageTechnology', label: 'Espionage' },
      { key: 'computerTechnology', label: 'Computer' },
      { key: 'astrophysics', label: 'Astrophysics' },
      { key: 'intergalacticResearchNetwork', label: 'IRN' },
      { key: 'gravitonTechnology', label: 'Graviton' },
    ],
  },
];

function levelColor(level: number): string {
  if (level === 0) return 'research-low';
  if (level < 8) return 'research-mid';
  return 'research-high';
}

export default function ResearchPanel(props: { research: APIResearch | null }) {
  return (
    <section class="research-panel">
      <h2>Research</h2>
      <Show when={props.research} fallback={<p class="empty">No research data</p>}>
        <div class="research-categories">
          {categories.map((cat) => (
            <div class="research-category">
              <h3>{cat.name}</h3>
              <div class="research-techs">
                {cat.techs.map((tech) => (
                  <div class={`research-tech ${levelColor(props.research![tech.key])}`}>
                    <span class="tech-name">{tech.label}</span>
                    <span class="tech-level">{props.research![tech.key]}</span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </Show>
    </section>
  );
}

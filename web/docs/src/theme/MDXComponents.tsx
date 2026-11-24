import MDXComponents from '@theme-original/MDXComponents';
import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

// Tabs e TabItem ficam disponíveis em qualquer página sem `import` —
// os exemplos multi-linguagem usam <Tabs groupId="linguagem">.
export default {
  ...MDXComponents,
  Tabs,
  TabItem,
};

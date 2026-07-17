// Mirror of the server-side password policy (internal/crypto/password.go) so
// the UI can give per-rule feedback before the server is ever hit — keep both
// in sync. The classes are unicode (\p{Lu}/\p{Ll}/\p{Nd}) to match Go's
// unicode.IsUpper/IsLower/IsDigit: with ASCII classes the two sides disagree
// on accented letters ("é" would count as the special character here while
// the server counts it as a lowercase letter).

export const PASSWORD_MIN_LENGTH = 8;

export type PasswordRule = {
  id: string;
  label: string;
  test: (password: string) => boolean;
};

export const PASSWORD_RULES: PasswordRule[] = [
  {
    id: 'length',
    label: `At least ${PASSWORD_MIN_LENGTH} characters`,
    test: password => password.length >= PASSWORD_MIN_LENGTH,
  },
  {
    id: 'uppercase',
    label: 'An uppercase letter',
    test: password => /\p{Lu}/u.test(password),
  },
  {
    id: 'lowercase',
    label: 'A lowercase letter',
    test: password => /\p{Ll}/u.test(password),
  },
  {
    id: 'digit',
    label: 'A digit',
    test: password => /\p{Nd}/u.test(password),
  },
  {
    id: 'special',
    label: 'A special character',
    test: password => /[^\p{Lu}\p{Ll}\p{Nd}]/u.test(password),
  },
];

export const isPasswordValid = (password: string) =>
  PASSWORD_RULES.every(rule => rule.test(password));

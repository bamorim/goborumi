package borumi

import (
	"fmt"
	"strings"
)

type Scene struct {
	Index  int    `json:"index,omitempty"`
	ID     string `json:"id"`
	Seq    string `json:"seq"`
	Name   string `json:"name"`
	Script string `json:"script"`
}

func ListScenes(dbPath string) ([]Scene, error) {
	rows, err := queryJSON(dbPath, `
SELECT
  id,
  seq,
  name,
  script
FROM scenes
ORDER BY CAST(seq AS INTEGER), seq;
`)
	if err != nil {
		return nil, err
	}

	scenes := make([]Scene, 0, len(rows))
	for i, row := range rows {
		scenes = append(scenes, Scene{
			Index:  i + 1,
			ID:     asString(row["id"]),
			Seq:    asString(row["seq"]),
			Name:   asString(row["name"]),
			Script: asString(row["script"]),
		})
	}
	return scenes, nil
}

func GetScene(dbPath string, sceneSelector string) (Scene, error) {
	sceneSelector = strings.TrimSpace(sceneSelector)
	quotedSelector := sqlQuote(sceneSelector)

	rows, err := queryJSON(dbPath, fmt.Sprintf(`
SELECT
  id,
  seq,
  name,
  script
FROM scenes
WHERE id = %s OR name = %s OR seq = %s
ORDER BY CASE
  WHEN id = %s THEN 0
  WHEN name = %s THEN 1
  WHEN seq = %s THEN 2
  ELSE 3
END;
`, quotedSelector, quotedSelector, quotedSelector, quotedSelector, quotedSelector, quotedSelector))
	if err != nil {
		return Scene{}, err
	}

	if len(rows) == 0 {
		return Scene{}, fmt.Errorf("scene not found: %q", sceneSelector)
	}

	if len(rows) > 1 {
		return Scene{}, fmt.Errorf("scene selector %q is ambiguous (%d matches). use scene id", sceneSelector, len(rows))
	}

	row := rows[0]
	scene := Scene{
		ID:     asString(row["id"]),
		Seq:    asString(row["seq"]),
		Name:   asString(row["name"]),
		Script: asString(row["script"]),
	}

	scene.Index, _ = lookupSceneIndex(dbPath, scene.ID)
	return scene, nil
}

func GetSceneByIndex(dbPath string, index int) (Scene, error) {
	if index < 1 {
		return Scene{}, fmt.Errorf("index must be >= 1")
	}

	scenes, err := ListScenes(dbPath)
	if err != nil {
		return Scene{}, err
	}

	if index > len(scenes) {
		return Scene{}, fmt.Errorf("scene index out of range: %d (max %d)", index, len(scenes))
	}

	return scenes[index-1], nil
}

func SetSceneScript(dbPath string, sceneID string, newScript string) (Scene, error) {
	query := fmt.Sprintf(
		"UPDATE scenes SET script = %s WHERE id = %s;",
		sqlQuote(newScript),
		sqlQuote(sceneID),
	)
	if err := execWrite(dbPath, query); err != nil {
		return Scene{}, err
	}

	return GetScene(dbPath, sceneID)
}

func lookupSceneIndex(dbPath string, sceneID string) (int, error) {
	rows, err := queryJSON(dbPath, fmt.Sprintf(`
SELECT idx
FROM (
  SELECT
    id,
    ROW_NUMBER() OVER (ORDER BY CAST(seq AS INTEGER), seq) AS idx
  FROM scenes
)
WHERE id = %s;
`, sqlQuote(sceneID)))
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, fmt.Errorf("scene id not found: %s", sceneID)
	}

	switch value := rows[0]["idx"].(type) {
	case float64:
		return int(value), nil
	case int:
		return value, nil
	default:
		return 0, fmt.Errorf("unexpected index value type: %T", value)
	}
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	if typed, ok := value.(string); ok {
		return typed
	}
	return fmt.Sprintf("%v", value)
}

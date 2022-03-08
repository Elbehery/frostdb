package columnstore

import (
	"context"
	"sort"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/parcaparquet"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
	"github.com/prometheus/prometheus/model/labels"
)

func ParcaProfilingTableConfig() *TableConfig {
	return NewTableConfig(
		parcaparquet.Schema(),
		8192, // 2^13
	)
}

func InsertProfileIntoTable(ctx context.Context, logger log.Logger, table *Table, ls labels.Labels, prof *parcaprofile.FlatProfile) (int, error) {
	// TODO all of this should be done in the flat profile
	// extraction in the first place. Also this `__name__` hack is
	// only here for backward compatibility while we finish up the
	// columnstore. This can be removed once the migration is
	// complete and the old storage is removed.
	lbls := make(labels.Labels, 0, len(ls)+1)
	found := false
	for _, l := range ls {
		if l.Name == "__name__" {
			found = true
			lbls = append(lbls, labels.Label{
				Name:  "__name__",
				Value: l.Value + "_" + prof.Meta.SampleType.Type + "_" + prof.Meta.SampleType.Unit,
			})
			continue
		}
		lbls = append(lbls, labels.Label{
			Name:  l.Name,
			Value: l.Value,
		})
	}
	if !found {
		lbls = append(lbls, labels.Label{
			Name:  "__name__",
			Value: prof.Meta.SampleType.Type + "_" + prof.Meta.SampleType.Unit,
		})
	}
	sort.Sort(lbls)

	rows := make(parcaparquet.Samples, 0, len(prof.FlatSamples))
	for _, s := range prof.FlatSamples {
		rows = append(rows, parcaparquet.Sample{
			SampleType: prof.Meta.SampleType.Type,
			SampleUnit: prof.Meta.SampleType.Unit,
			PeriodType: prof.Meta.PeriodType.Type,
			PeriodUnit: prof.Meta.PeriodType.Unit,
			Labels:     lbls,
			Stacktrace: extractLocationIDs(s.Location),
			Timestamp:  prof.Meta.Timestamp,
			Duration:   prof.Meta.Duration,
			Period:     prof.Meta.Period,
			Value:      s.Value,
		})
	}

	level.Debug(logger).Log("msg", "writing sample", "label_set", ls.String(), "timestamp", prof.Meta.Timestamp)

	buf, err := rows.ToBuffer(table.Schema())
	if err != nil {
		return 0, err
	}

	buf.Sort()
	return len(prof.FlatSamples), table.Insert(buf)
}

func extractLocationIDs(locs []*metastore.Location) []uuid.UUID {
	res := make([]uuid.UUID, 0, len(locs))
	for i := len(locs) - 1; i >= 0; i-- {
		res = append(res, locs[i].ID)
	}
	return res
}

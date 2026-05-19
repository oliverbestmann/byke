
static const int X = 0;
static const int Y = 1;
static const int Z = 2;
static const int W = 3;

struct vec4 {
  float x, y, z, w;
};

struct mat4 {
  float columns[4][4];
};

void mat4f_mul_assign(struct mat4 *restrict m, struct mat4 *restrict o) {
  struct mat4 morg = *m;

  for (int i = 0; i < 4; i++) {
    m->columns[i][X] = morg.columns[X][X] * o->columns[i][X] +
                       morg.columns[Y][X] * o->columns[i][Y] +
                       morg.columns[Z][X] * o->columns[i][Z] +
                       morg.columns[W][X] * o->columns[i][W];

    m->columns[i][Y] = morg.columns[X][Y] * o->columns[i][X] +
                       morg.columns[Y][Y] * o->columns[i][Y] +
                       morg.columns[Z][Y] * o->columns[i][Z] +
                       morg.columns[W][Y] * o->columns[i][W];

    m->columns[i][Z] = morg.columns[X][Z] * o->columns[i][X] +
                       morg.columns[Y][Z] * o->columns[i][Y] +
                       morg.columns[Z][Z] * o->columns[i][Z] +
                       morg.columns[W][Z] * o->columns[i][W];

    m->columns[i][W] = morg.columns[X][W] * o->columns[i][X] +
                       morg.columns[Y][W] * o->columns[i][Y] +
                       morg.columns[Z][W] * o->columns[i][Z] +
                       morg.columns[W][W] * o->columns[i][W];
  }
}

void mat4f_fast_translate(struct mat4 *m, float x, float y, float z) {
  for (int i = 0; i < 4; i++) {
    m->columns[W][i] = m->columns[X][i] * x + //
                       m->columns[Y][i] * y + //
                       m->columns[Z][i] * z + //
                       m->columns[W][i];
  }
}

void mat4f_fast_scale(struct mat4 *m, float x, float y, float z) {
  m->columns[0][X] = m->columns[0][X] * x;
  m->columns[0][Y] = m->columns[0][Y] * x;
  m->columns[0][Z] = m->columns[0][Z] * x;
  m->columns[0][W] = m->columns[0][W] * x;

  m->columns[1][X] = m->columns[1][X] * y;
  m->columns[1][Y] = m->columns[1][Y] * y;
  m->columns[1][Z] = m->columns[1][Z] * y;
  m->columns[1][W] = m->columns[1][W] * y;

  m->columns[2][X] = m->columns[2][X] * z;
  m->columns[2][Y] = m->columns[2][Y] * z;
  m->columns[2][Z] = m->columns[2][Z] * z;
  m->columns[2][W] = m->columns[2][W] * z;
}
